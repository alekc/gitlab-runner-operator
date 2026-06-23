/*
Copyright 2020 Alexander Chernov

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/api"
	"gitlab.k8s.alekc.dev/internal/crud"
	"gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// secretTokenKey is the data key read from a referenced token secret.
	secretTokenKey = "token"
	// tokenRefreshThreshold: a managed token within this window of expiry is
	// refreshed in place (no recreate).
	tokenRefreshThreshold = 24 * time.Hour
	// tokenResyncLead: managed runners requeue this long before token expiry so
	// the refresh happens ahead of time.
	tokenResyncLead = 12 * time.Hour
	// maxResyncInterval caps the steady-state requeue for managed runners.
	maxResyncInterval = time.Hour
	// minResyncInterval floors it so the controller never busy-loops.
	minResyncInterval = time.Minute

	maxDeleteAttempts        = 5
	deleteAttemptsAnnotation = "gitlab.k8s.alekc.dev/delete-attempts"
)

// resolveToken returns the inline token if set, otherwise the value of the
// "token" key from the named secret in namespace. It returns an empty string
// (no error) when neither source is configured.
func resolveToken(ctx context.Context, cl client.Client, namespace, inline, secretName string) (string, error) {
	if inline != "" {
		return inline, nil
	}
	if secretName == "" {
		return "", nil
	}
	var secret corev1.Secret
	key := client.ObjectKey{Namespace: namespace, Name: secretName}
	if err := cl.Get(ctx, key, &secret); err != nil {
		return "", fmt.Errorf("cannot read secret %q: %w", secretName, err)
	}
	return string(secret.Data[secretTokenKey]), nil
}

// ensureRunners makes sure every runner unit is authenticated and returns the
// resolved authentication tokens keyed by entry name (used to render the config
// Secret). It also returns a RequeueAfter so managed-runner token expiry is
// re-checked even without a spec change. The token is never stored in status;
// managed tokens are persisted in the config Secret and recovered from there.
func ensureRunners(ctx context.Context, cl client.Client, statusW client.StatusWriter, injected api.GitlabClient, obj types.RunnerInfo, logger logr.Logger) (map[string]string, time.Duration, error) {
	namespace := obj.GetNamespace()
	existing, err := crud.ExistingConfigTokens(ctx, cl, namespace, obj.ChildName())
	if err != nil {
		return nil, 0, fmt.Errorf("cannot read existing config secret: %w", err)
	}

	tokens := map[string]string{}
	var soonestExpiry *time.Time
	for _, reg := range obj.RegistrationConfig() {
		reg := reg
		if !reg.Auth.IsManaged() {
			token, err := resolveToken(ctx, cl, namespace, reg.Auth.AuthenticationToken, reg.Auth.AuthenticationTokenSecret)
			if err != nil {
				return nil, 0, err
			}
			if token == "" {
				return nil, 0, fmt.Errorf("runner %q requires authentication_token or authentication_token_secret", reg.Name)
			}
			tokens[reg.Name] = token
			logger.Info("using provided authentication token", "name", reg.Name)
			continue
		}

		token, expiry, err := ensureManagedRunner(ctx, cl, statusW, injected, obj, &reg, existing[reg.Name], logger)
		if err != nil {
			return nil, 0, err
		}
		tokens[reg.Name] = token
		if expiry != nil && (soonestExpiry == nil || expiry.Before(*soonestExpiry)) {
			soonestExpiry = expiry
		}
	}
	return tokens, resyncAfter(soonestExpiry), nil
}

// resyncAfter computes how long until the controller should re-reconcile a
// managed runner to refresh its token ahead of expiry. Zero means no managed
// expiry to watch.
func resyncAfter(soonestExpiry *time.Time) time.Duration {
	if soonestExpiry == nil {
		return 0
	}
	d := time.Until(soonestExpiry.Add(-tokenResyncLead))
	if d < minResyncInterval {
		d = minResyncInterval
	}
	if d > maxResyncInterval {
		d = maxResyncInterval
	}
	return d
}

// ensureManagedRunner reconciles a single managed runner against GitLab and
// returns its current authentication token and expiry.
func ensureManagedRunner(ctx context.Context, cl client.Client, statusW client.StatusWriter, injected api.GitlabClient, obj types.RunnerInfo, reg *v1beta2.GitlabRegInfo, recoveredToken string, logger logr.Logger) (string, *time.Time, error) {
	desiredHash := reg.Auth.CreateOptions.Hash()

	gitlabClient := func() (api.GitlabClient, error) {
		if injected != nil {
			return injected, nil
		}
		accessToken, err := resolveToken(ctx, cl, obj.GetNamespace(), reg.Auth.AccessToken, reg.Auth.AccessTokenSecret)
		if err != nil {
			return nil, err
		}
		if accessToken == "" {
			return nil, fmt.Errorf("managed runner %q requires access_token or access_token_secret", reg.Name)
		}
		return api.NewGitlabClient(accessToken, reg.GitlabUrl)
	}

	switch {
	case reg.RunnerID == 0:
		gc, err := gitlabClient()
		if err != nil {
			return "", nil, err
		}
		created, err := gc.CreateRunner(*reg.Auth.CreateOptions)
		if err != nil {
			return "", nil, fmt.Errorf("cannot create runner %q on gitlab: %w", reg.Name, err)
		}
		return persistRunner(ctx, statusW, obj, reg, created, desiredHash, logger, "created")

	case reg.RegistrationHash != desiredHash:
		gc, err := gitlabClient()
		if err != nil {
			return "", nil, err
		}
		// Recreate: delete the old runner FIRST. If that fails do NOT create a
		// replacement (it would orphan the old runner on GitLab); return the
		// error and let the reconcile retry.
		if err := gc.DeleteRunner(reg.RunnerID); err != nil {
			return "", nil, fmt.Errorf("cannot delete stale runner %q (id %d) before recreate: %w", reg.Name, reg.RunnerID, err)
		}
		created, err := gc.CreateRunner(*reg.Auth.CreateOptions)
		if err != nil {
			return "", nil, fmt.Errorf("cannot recreate runner %q on gitlab: %w", reg.Name, err)
		}
		return persistRunner(ctx, statusW, obj, reg, created, desiredHash, logger, "recreated")

	case reg.TokenExpiresAt != nil && time.Until(reg.TokenExpiresAt.Time) < tokenRefreshThreshold:
		gc, err := gitlabClient()
		if err != nil {
			return "", nil, err
		}
		refreshed, err := gc.RefreshToken(reg.RunnerID)
		if err != nil {
			return "", nil, fmt.Errorf("cannot refresh token for runner %q: %w", reg.Name, err)
		}
		return persistRunner(ctx, statusW, obj, reg, refreshed, desiredHash, logger, "token refreshed")

	default:
		// Steady state. Recover the token from the existing Secret; if it is
		// gone, reset it. Otherwise verify it is still accepted by GitLab.
		gc, err := gitlabClient()
		if err != nil {
			return "", nil, err
		}
		if recoveredToken == "" {
			refreshed, err := gc.RefreshToken(reg.RunnerID)
			if err != nil {
				return "", nil, fmt.Errorf("cannot recover token for runner %q: %w", reg.Name, err)
			}
			return persistRunner(ctx, statusW, obj, reg, refreshed, desiredHash, logger, "token recovered")
		}
		valid, err := gc.VerifyToken(recoveredToken)
		if err != nil {
			return "", nil, fmt.Errorf("cannot verify token for runner %q: %w", reg.Name, err)
		}
		if !valid {
			logger.Info("managed runner token rejected by gitlab, recreating", "name", reg.Name, "id", reg.RunnerID)
			if err := gc.DeleteRunner(reg.RunnerID); err != nil {
				return "", nil, fmt.Errorf("cannot delete invalid runner %q (id %d) before recreate: %w", reg.Name, reg.RunnerID, err)
			}
			created, err := gc.CreateRunner(*reg.Auth.CreateOptions)
			if err != nil {
				return "", nil, fmt.Errorf("cannot recreate runner %q after invalid token: %w", reg.Name, err)
			}
			return persistRunner(ctx, statusW, obj, reg, created, desiredHash, logger, "recreated")
		}
		var expiry *time.Time
		if reg.TokenExpiresAt != nil {
			t := reg.TokenExpiresAt.Time
			expiry = &t
		}
		return recoveredToken, expiry, nil
	}
}

// persistRunner records the create/recreate/refresh result in status and writes
// it IMMEDIATELY, so a crash before the deferred status update cannot lose the
// runner id (which would orphan the GitLab runner and duplicate it next pass).
func persistRunner(ctx context.Context, statusW client.StatusWriter, obj types.RunnerInfo, reg *v1beta2.GitlabRegInfo, created api.CreatedRunner, hash string, logger logr.Logger, verb string) (string, *time.Time, error) {
	reg.RunnerID = created.ID
	reg.AuthToken = created.Token
	reg.TokenExpiresAt = created.TokenExpiresAt
	reg.RegistrationHash = hash
	obj.StoreRunnerRegistration(*reg)
	if err := obj.UpdateStatus(ctx, statusW); err != nil {
		return "", nil, fmt.Errorf("cannot persist status for runner %q: %w", reg.Name, err)
	}
	logger.Info("managed runner "+verb, "name", reg.Name, "id", reg.RunnerID)

	var expiry *time.Time
	if created.TokenExpiresAt != nil {
		t := created.TokenExpiresAt.Time
		expiry = &t
	}
	return created.Token, expiry, nil
}

// removeManagedRunners deletes managed runners from GitLab when the owning
// object is being deleted. A runner that is already gone counts as deleted. It
// returns the first error encountered so the caller can retry before dropping
// the finalizer (bring-your-own-token units are never touched).
func removeManagedRunners(ctx context.Context, cl client.Client, injected api.GitlabClient, obj types.RunnerInfo, logger logr.Logger) error {
	namespace := obj.GetNamespace()
	var firstErr error
	for _, reg := range obj.RegistrationConfig() {
		if !reg.Auth.IsManaged() || reg.RunnerID == 0 {
			continue
		}
		gc := injected
		if gc == nil {
			accessToken, err := resolveToken(ctx, cl, namespace, reg.Auth.AccessToken, reg.Auth.AccessTokenSecret)
			if err != nil {
				logger.Error(err, "cannot resolve access token for runner deletion", "name", reg.Name)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			if gc, err = api.NewGitlabClient(accessToken, reg.GitlabUrl); err != nil {
				logger.Error(err, "cannot build gitlab client for runner deletion", "name", reg.Name)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
		}
		if err := gc.DeleteRunner(reg.RunnerID); err != nil {
			// If the runner is already gone, treat the deletion as complete.
			if exists, exErr := gc.RunnerExists(reg.RunnerID); exErr == nil && !exists {
				continue
			}
			logger.Error(err, "cannot delete runner from gitlab", "name", reg.Name, "id", reg.RunnerID)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// finalizeDeletion removes managed runners from GitLab, then the finalizer. On
// failure it keeps the finalizer and requeues so the runner is not orphaned,
// up to maxDeleteAttempts; after that it logs loudly and removes the finalizer
// anyway so the object never wedges.
func finalizeDeletion(ctx context.Context, cl client.Client, injected api.GitlabClient, obj types.RunnerInfo, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("runner is being deleted")
	if err := removeManagedRunners(ctx, cl, injected, obj, logger); err != nil {
		attempts := deleteAttempts(obj) + 1
		setDeleteAttempts(obj, attempts)
		if uErr := obj.Update(ctx, cl); uErr != nil {
			logger.Error(uErr, "cannot persist delete-attempt counter")
		}
		if attempts < maxDeleteAttempts {
			logger.Error(err, "failed to delete managed runner(s) from gitlab, will retry", "attempt", attempts)
			return resultRequeueAfterDefaultTimeout, err
		}
		logger.Error(err, "giving up deleting managed runner(s) from gitlab after max attempts; removing finalizer, gitlab-side runners may be orphaned", "attempts", attempts)
	}

	obj.RemoveFinalizer()
	if err := obj.Update(ctx, cl); err != nil {
		logger.Error(err, "cannot remove finalizer")
		return resultRequeueAfterDefaultTimeout, err
	}
	return resultRequeueNow, nil
}

func deleteAttempts(obj types.RunnerInfo) int {
	if v, ok := obj.GetAnnotations()[deleteAttemptsAnnotation]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

func setDeleteAttempts(obj types.RunnerInfo, n int) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[deleteAttemptsAnnotation] = strconv.Itoa(n)
	obj.SetAnnotations(annotations)
}
