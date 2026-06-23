/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/internal/api"
	"gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// secretTokenKey is the data key read from a referenced secret.
const secretTokenKey = "token"

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

// reconcileRegistration ensures every runner unit is authenticated. Managed
// units (access token + create options) are created via the GitLab API and
// recreated when their create options change; bring-your-own-token units simply
// have their token resolved into status. Results are written back through
// StoreRunnerRegistration.
func reconcileRegistration(ctx context.Context, cl client.Client, injected api.GitlabClient, obj types.RunnerInfo, logger logr.Logger) error {
	namespace := obj.GetNamespace()
	for _, reg := range obj.RegistrationConfig() {
		if reg.Auth.IsManaged() {
			gc := injected
			if gc == nil {
				accessToken, err := resolveToken(ctx, cl, namespace, reg.Auth.AccessToken, reg.Auth.AccessTokenSecret)
				if err != nil {
					return err
				}
				if accessToken == "" {
					return fmt.Errorf("managed runner %q requires access_token or access_token_secret", reg.Name)
				}
				if gc, err = api.NewGitlabClient(accessToken, reg.GitlabUrl); err != nil {
					return err
				}
			}

			// A managed runner whose create options changed is recreated: drop
			// the stale runner first (best effort), then create the new one.
			if reg.RunnerID != 0 {
				if err := gc.DeleteRunner(reg.RunnerID); err != nil {
					logger.Error(err, "cannot delete stale runner from gitlab", "id", reg.RunnerID)
				}
			}
			created, err := gc.CreateRunner(*reg.Auth.CreateOptions)
			if err != nil {
				return fmt.Errorf("cannot create runner %q on gitlab: %w", reg.Name, err)
			}
			reg.RunnerID = created.ID
			reg.AuthToken = created.Token
			reg.TokenExpiresAt = created.TokenExpiresAt
			reg.RegistrationHash = reg.Auth.CreateOptions.Hash()
			logger.Info("created managed runner on gitlab", "name", reg.Name, "id", reg.RunnerID)
		} else {
			token, err := resolveToken(ctx, cl, namespace, reg.Auth.AuthenticationToken, reg.Auth.AuthenticationTokenSecret)
			if err != nil {
				return err
			}
			if token == "" {
				return fmt.Errorf("runner %q requires authentication_token or authentication_token_secret", reg.Name)
			}
			reg.AuthToken = token
			logger.Info("using provided authentication token", "name", reg.Name)
		}
		obj.StoreRunnerRegistration(reg)
	}
	return nil
}

// removeManagedRunners deletes managed runners from GitLab when the owning
// object is being deleted. Bring-your-own-token units are left untouched.
// Failures are logged, not fatal: deletion must always be able to complete.
func removeManagedRunners(ctx context.Context, cl client.Client, injected api.GitlabClient, obj types.RunnerInfo, logger logr.Logger) {
	namespace := obj.GetNamespace()
	for _, reg := range obj.RegistrationConfig() {
		if !reg.Auth.IsManaged() || reg.RunnerID == 0 {
			continue
		}
		gc := injected
		if gc == nil {
			accessToken, err := resolveToken(ctx, cl, namespace, reg.Auth.AccessToken, reg.Auth.AccessTokenSecret)
			if err != nil {
				logger.Error(err, "cannot resolve access token for runner deletion", "name", reg.Name)
				continue
			}
			if gc, err = api.NewGitlabClient(accessToken, reg.GitlabUrl); err != nil {
				logger.Error(err, "cannot build gitlab client for runner deletion", "name", reg.Name)
				continue
			}
		}
		if err := gc.DeleteRunner(reg.RunnerID); err != nil {
			logger.Error(err, "cannot delete runner from gitlab", "name", reg.Name, "id", reg.RunnerID)
		}
	}
}
