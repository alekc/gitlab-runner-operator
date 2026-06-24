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

package v1beta2

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager registers the defaulting and validating webhooks for
// the MultiRunner type with the manager.
func (r *MultiRunner) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(&MultiRunnerWebhook{}).
		WithValidator(&MultiRunnerWebhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-gitlab-k8s-alekc-dev-v1beta2-multirunner,mutating=true,failurePolicy=fail,sideEffects=None,groups=gitlab.k8s.alekc.dev,resources=multirunners,verbs=create;update,versions=v1beta2,name=mmultirunner.kb.io,admissionReviewVersions={v1,v1beta1}
// +kubebuilder:webhook:path=/validate-gitlab-k8s-alekc-dev-v1beta2-multirunner,mutating=false,failurePolicy=fail,sideEffects=None,groups=gitlab.k8s.alekc.dev,resources=multirunners,verbs=create;update,versions=v1beta2,name=vmultirunner.kb.io,admissionReviewVersions={v1,v1beta1}

// MultiRunnerWebhook implements the controller-runtime defaulting and
// validating webhook interfaces for the MultiRunner type.
type MultiRunnerWebhook struct{}

var (
	_ admission.Defaulter[*MultiRunner] = &MultiRunnerWebhook{}
	_ admission.Validator[*MultiRunner] = &MultiRunnerWebhook{}
)

// Default applies sane defaults to a MultiRunner before it is persisted.
func (w *MultiRunnerWebhook) Default(_ context.Context, r *MultiRunner) error {
	if r.Spec.GitlabInstanceURL == "" {
		r.Spec.GitlabInstanceURL = "https://gitlab.com/"
	}
	return nil
}

// ValidateCreate validates every entry's auth and entry-name uniqueness.
func (w *MultiRunnerWebhook) ValidateCreate(_ context.Context, r *MultiRunner) (admission.Warnings, error) {
	return nil, validateEntries(r)
}

// ValidateUpdate re-runs entry validation against the updated object.
func (w *MultiRunnerWebhook) ValidateUpdate(_ context.Context, _, newObj *MultiRunner) (admission.Warnings, error) {
	return nil, validateEntries(newObj)
}

// ValidateDelete is a no-op placeholder kept for future validation rules.
func (w *MultiRunnerWebhook) ValidateDelete(_ context.Context, _ *MultiRunner) (admission.Warnings, error) {
	return nil, nil
}

func validateEntries(r *MultiRunner) error {
	if len(r.Spec.Entries) == 0 {
		return fmt.Errorf("a multirunner requires at least one entry")
	}
	seen := make(map[string]struct{}, len(r.Spec.Entries))
	for i, entry := range r.Spec.Entries {
		if entry.Name == "" {
			return fmt.Errorf("entries[%d]: name is required", i)
		}
		if _, dup := seen[entry.Name]; dup {
			return fmt.Errorf("duplicate entry name %q", entry.Name)
		}
		seen[entry.Name] = struct{}{}
		if err := entry.Authentication.Validate(); err != nil {
			return fmt.Errorf("entry %q: %w", entry.Name, err)
		}
		if err := validateKubernetesExecutor(&r.Spec.Entries[i].ExecutorConfig); err != nil {
			return fmt.Errorf("entry %q: %w", entry.Name, err)
		}
	}
	return nil
}

// validateKubernetesExecutor rejects executor settings that make the build
// namespace dynamic. The operator pre-provisions namespaced RBAC for the
// runner ServiceAccount, so it cannot cover a namespace chosen at job time;
// supporting these would require cluster-scoped RBAC the operator does not
// grant. Failing at admission is clearer than a forbidden error at job time.
func validateKubernetesExecutor(cfg *KubernetesConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.NamespacePerJob {
		return fmt.Errorf("namespace_per_job is not supported: it would need cluster-scoped RBAC the operator does not grant")
	}
	if cfg.NamespaceOverwriteAllowed != "" {
		return fmt.Errorf("namespace_overwrite_allowed is not supported: the build namespace must be static so the operator can provision RBAC for it")
	}
	return nil
}
