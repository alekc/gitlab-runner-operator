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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// defaultSecretTokenKey is the Secret data key read when a TokenSource's
	// SecretKeyRef does not specify one.
	defaultSecretTokenKey = "token"
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

// resolveTokenSource resolves a TokenSource to its literal token. An inline
// value wins; otherwise the SecretKeyRef's key (defaulting to "token") is read
// from a Secret in namespace. When the ref is marked optional a missing secret
// or key yields an empty token instead of an error. A nil or empty source
// returns an empty string with no error.
func resolveTokenSource(ctx context.Context, cl client.Client, namespace string, src *v1beta2.TokenSource) (string, error) {
	if src == nil {
		return "", nil
	}
	if src.Value != "" {
		return src.Value, nil
	}
	ref := src.SecretKeyRef
	if ref == nil {
		return "", nil
	}
	optional := ref.Optional != nil && *ref.Optional

	var secret corev1.Secret
	key := client.ObjectKey{Namespace: namespace, Name: ref.Name}
	if err := cl.Get(ctx, key, &secret); err != nil {
		if optional && apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("cannot read secret %q: %w", ref.Name, err)
	}

	dataKey := ref.Key
	if dataKey == "" {
		dataKey = defaultSecretTokenKey
	}
	value, ok := secret.Data[dataKey]
	if !ok {
		if optional {
			return "", nil
		}
		return "", fmt.Errorf("secret %q has no key %q", ref.Name, dataKey)
	}
	return string(value), nil
}

// resolveCABundle resolves a CASource to its PEM bytes, read from the named key
// (defaulting to v1beta2.DefaultCAKey) of a Secret or ConfigMap in namespace. A
// nil or empty source returns nil with no error.
func resolveCABundle(ctx context.Context, cl client.Client, namespace string, src *v1beta2.CASource) ([]byte, error) {
	if !src.IsSet() {
		return nil, nil
	}
	switch {
	case src.SecretKeyRef != nil:
		ref := src.SecretKeyRef
		var secret corev1.Secret
		if err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, &secret); err != nil {
			return nil, fmt.Errorf("cannot read CA secret %q: %w", ref.Name, err)
		}
		key := ref.Key
		if key == "" {
			key = v1beta2.DefaultCAKey
		}
		data, ok := secret.Data[key]
		if !ok {
			return nil, fmt.Errorf("CA secret %q has no key %q", ref.Name, key)
		}
		return data, nil
	case src.ConfigMapKeyRef != nil:
		ref := src.ConfigMapKeyRef
		var cm corev1.ConfigMap
		if err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, &cm); err != nil {
			return nil, fmt.Errorf("cannot read CA configmap %q: %w", ref.Name, err)
		}
		key := ref.Key
		if key == "" {
			key = v1beta2.DefaultCAKey
		}
		if data, ok := cm.Data[key]; ok {
			return []byte(data), nil
		}
		if data, ok := cm.BinaryData[key]; ok {
			return data, nil
		}
		return nil, fmt.Errorf("CA configmap %q has no key %q", ref.Name, key)
	}
	return nil, nil
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
			token, err := resolveTokenSource(ctx, cl, namespace, reg.Auth.Token)
			if err != nil {
				return nil, 0, err
			}
			if token == "" {
				return nil, 0, fmt.Errorf("runner %q requires a token (value or secret_key_ref)", reg.Name)
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
		accessToken, err := resolveTokenSource(ctx, cl, obj.GetNamespace(), reg.Auth.AccessToken)
		if err != nil {
			return nil, err
		}
		if accessToken == "" {
			return nil, fmt.Errorf("managed runner %q requires an access_token (value or secret_key_ref)", reg.Name)
		}
		caPEM, err := resolveCABundle(ctx, cl, obj.GetNamespace(), reg.CACertificate)
		if err != nil {
			return nil, err
		}
		return api.NewGitlabClient(accessToken, reg.GitlabUrl, caPEM)
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
		return recreateManagedRunner(ctx, statusW, gitlabClient, obj, reg, recoveredToken, desiredHash, logger, "recreated (create options changed)")

	case reg.TokenExpiresAt != nil && time.Until(reg.TokenExpiresAt.Time) < tokenRefreshThreshold:
		// GitLab has no reset-by-token endpoint, and resetting by id needs the
		// api scope; recreate instead so create_runner stays sufficient.
		return recreateManagedRunner(ctx, statusW, gitlabClient, obj, reg, recoveredToken, desiredHash, logger, "recreated (token near expiry)")

	default:
		// Steady state. Without the token we can neither verify nor delete it by
		// token, so recreate (the old runner may be orphaned). Otherwise verify
		// the token and recreate only if GitLab has rejected it.
		if recoveredToken == "" {
			return recreateManagedRunner(ctx, statusW, gitlabClient, obj, reg, "", desiredHash, logger, "recreated (token missing from config secret)")
		}
		gc, err := gitlabClient()
		if err != nil {
			return "", nil, err
		}
		valid, err := gc.VerifyToken(recoveredToken)
		if err != nil {
			return "", nil, fmt.Errorf("cannot verify token for runner %q: %w", reg.Name, err)
		}
		if !valid {
			logger.Info("managed runner token rejected by gitlab, recreating", "name", reg.Name, "id", reg.RunnerID)
			return recreateManagedRunner(ctx, statusW, gitlabClient, obj, reg, recoveredToken, desiredHash, logger, "recreated (token rejected)")
		}
		var expiry *time.Time
		if reg.TokenExpiresAt != nil {
			t := reg.TokenExpiresAt.Time
			expiry = &t
		}
		return recoveredToken, expiry, nil
	}
}

// recreateManagedRunner deletes the existing managed runner from GitLab using
// its own authentication token (DELETE /runners by token, which needs no api
// scope) and creates a fresh one. The delete happens FIRST so a failure does
// not leave two runners registered. When the old token is unavailable the old
// runner cannot be deleted (that would require the api scope); a new runner is
// created and the old one is logged as possibly orphaned.
func recreateManagedRunner(ctx context.Context, statusW client.StatusWriter, gitlabClient func() (api.GitlabClient, error), obj types.RunnerInfo, reg *v1beta2.GitlabRegInfo, oldToken, desiredHash string, logger logr.Logger, verb string) (string, *time.Time, error) {
	gc, err := gitlabClient()
	if err != nil {
		return "", nil, err
	}
	switch {
	case oldToken != "":
		if err := gc.DeleteRunner(oldToken); err != nil {
			// Fall back to delete-by-id with the operator access token (needs the
			// api scope) before giving up; do not create a replacement while the
			// old runner may still be registered.
			logger.Info("delete by token failed during recreate, falling back to delete by id", "name", reg.Name, "error", err.Error())
			if err := gc.DeleteRunnerByID(reg.RunnerID); err != nil {
				return "", nil, fmt.Errorf("cannot delete stale runner %q before recreate (by token and by id): %w", reg.Name, err)
			}
		}
	case reg.RunnerID != 0:
		// No token to delete by; try the access-token by-id fallback. If it also
		// fails (token lacks api), proceed and log the old runner as possibly
		// orphaned rather than blocking the recreate forever.
		if err := gc.DeleteRunnerByID(reg.RunnerID); err != nil {
			logger.Info("old managed runner token unavailable and delete by id failed; creating a new runner, the old one may be orphaned on gitlab", "name", reg.Name, "old_id", reg.RunnerID, "error", err.Error())
		}
	}
	created, err := gc.CreateRunner(*reg.Auth.CreateOptions)
	if err != nil {
		return "", nil, fmt.Errorf("cannot recreate runner %q on gitlab: %w", reg.Name, err)
	}
	return persistRunner(ctx, statusW, obj, reg, created, desiredHash, logger, verb)
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
// object is being deleted, using each runner's own authentication token
// recovered from the config Secret. Deleting by token needs no access token and
// no api scope, and the config Secret still exists at finalization (the
// finalizer blocks the object's, and so the Secret's, garbage collection). A
// runner whose token cannot be recovered cannot be deleted and is logged as
// possibly orphaned. Returns the first hard error so the caller can retry before
// dropping the finalizer (bring-your-own-token units are never touched).
func removeManagedRunners(ctx context.Context, cl client.Client, injected api.GitlabClient, obj types.RunnerInfo, logger logr.Logger) error {
	tokens, err := crud.ExistingConfigTokens(ctx, cl, obj.GetNamespace(), obj.ChildName())
	if err != nil {
		return fmt.Errorf("cannot read config secret to recover runner tokens: %w", err)
	}
	var firstErr error
	for _, reg := range obj.RegistrationConfig() {
		if !reg.Auth.IsManaged() || reg.RunnerID == 0 {
			continue
		}
		if err := deleteManagedRunner(ctx, cl, injected, obj.GetNamespace(), reg, tokens[reg.Name], logger); err != nil {
			logger.Error(err, "cannot delete runner from gitlab", "name", reg.Name, "id", reg.RunnerID)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// deleteManagedRunner removes one managed runner from GitLab, preferring its own
// authentication token (DELETE /runners by token, which needs no access-token
// scope) and falling back to the operator access token by runner id (DELETE
// /runners/:id, which needs the api scope) when the token is missing or the
// token-based delete is rejected. When neither path succeeds the runner may be
// left orphaned and an error is returned so the finalizer can retry.
func deleteManagedRunner(ctx context.Context, cl client.Client, injected api.GitlabClient, namespace string, reg v1beta2.GitlabRegInfo, token string, logger logr.Logger) error {
	caPEM, err := resolveCABundle(ctx, cl, namespace, reg.CACertificate)
	if err != nil {
		return err
	}
	// Preferred path: delete by the runner's own token (no access-token scope).
	if token != "" {
		gc := injected
		if gc == nil {
			c, err := api.NewGitlabClient("", reg.GitlabUrl, caPEM)
			if err != nil {
				return err
			}
			gc = c
		}
		err := gc.DeleteRunner(token)
		if err == nil {
			return nil
		}
		logger.Info("delete by token failed, falling back to the operator access token (delete by id, needs api scope)", "name", reg.Name, "error", err.Error())
	} else {
		logger.Info("no runner token in config secret, falling back to the operator access token (delete by id, needs api scope)", "name", reg.Name, "id", reg.RunnerID)
	}

	// Fallback: delete by id with the operator access token (needs api scope).
	gc := injected
	if gc == nil {
		accessToken, err := resolveTokenSource(ctx, cl, namespace, reg.Auth.AccessToken)
		if err != nil {
			return err
		}
		if accessToken == "" {
			return fmt.Errorf("runner %q: no usable runner token and no access_token for the delete-by-id fallback; it may be orphaned on gitlab", reg.Name)
		}
		c, err := api.NewGitlabClient(accessToken, reg.GitlabUrl, caPEM)
		if err != nil {
			return err
		}
		gc = c
	}
	if err := gc.DeleteRunnerByID(reg.RunnerID); err != nil {
		return fmt.Errorf("runner %q: delete by id fallback failed: %w", reg.Name, err)
	}
	return nil
}

// finalizeDeletion removes managed runners from GitLab, then the finalizer. On
// failure it keeps the finalizer and requeues so the runner is not orphaned,
// up to maxDeleteAttempts; after that it logs loudly and removes the finalizer
// anyway so the object never wedges.
func finalizeDeletion(ctx context.Context, cl client.Client, apiReader client.Reader, injected api.GitlabClient, obj types.RunnerInfo, logger logr.Logger) (ctrl.Result, error) {
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

	// Clean up RBAC the operator created in build namespaces other than the
	// runner's own; those lack owner references so Kubernetes will not garbage
	// collect them. Same-namespace RBAC is owner-referenced and collected on
	// object deletion. Keep the finalizer and requeue on failure.
	if err := crud.DeleteRBACExcept(ctx, cl, apiReader, obj, []string{obj.GetNamespace()}, logger); err != nil {
		logger.Error(err, "cannot clean up cross-namespace runner RBAC, will retry")
		return resultRequeueAfterDefaultTimeout, err
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
