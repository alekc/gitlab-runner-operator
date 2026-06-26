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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager registers the defaulting and validating webhooks for
// the Runner type with the manager. allowedBuildNamespaces lists the namespaces
// (besides a runner's own) where executor RBAC may be provisioned.
func (r *Runner) SetupWebhookWithManager(mgr ctrl.Manager, allowedBuildNamespaces []string) error {
	w := &RunnerWebhook{AllowedBuildNamespaces: allowedBuildNamespaces}
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-gitlab-k8s-alekc-dev-v1beta2-runner,mutating=true,failurePolicy=fail,sideEffects=None,groups=gitlab.k8s.alekc.dev,resources=runners,verbs=create;update,versions=v1beta2,name=mrunner.kb.io,admissionReviewVersions={v1,v1beta1}
// +kubebuilder:webhook:path=/validate-gitlab-k8s-alekc-dev-v1beta2-runner,mutating=false,failurePolicy=fail,sideEffects=None,groups=gitlab.k8s.alekc.dev,resources=runners,verbs=create;update,versions=v1beta2,name=vrunner.kb.io,admissionReviewVersions={v1,v1beta1}

// RunnerWebhook implements the controller-runtime defaulting and validating
// webhook interfaces for the Runner type. controller-runtime 0.19+ moved these
// off the API type onto a dedicated handler implementing the generic
// Defaulter / Validator interfaces.
type RunnerWebhook struct {
	// AllowedBuildNamespaces are namespaces (besides a runner's own) the
	// operator may provision executor RBAC in. Empty means own-namespace only;
	// "*" allows any namespace.
	AllowedBuildNamespaces []string
}

var (
	_ admission.Defaulter[*Runner] = &RunnerWebhook{}
	_ admission.Validator[*Runner] = &RunnerWebhook{}
)

// Default applies sane defaults to a Runner before it is persisted.
func (w *RunnerWebhook) Default(_ context.Context, r *Runner) error {
	if r.Spec.GitlabInstanceURL == "" {
		r.Spec.GitlabInstanceURL = "https://gitlab.com/"
	}
	return nil
}

// ValidateCreate enforces that exactly one auth mode is configured and that the
// executor namespacing is supportable.
func (w *RunnerWebhook) ValidateCreate(_ context.Context, r *Runner) (admission.Warnings, error) {
	if err := r.Spec.Authentication.Validate(); err != nil {
		return nil, err
	}
	if err := r.Spec.CACertificate.Validate(); err != nil {
		return nil, err
	}
	return nil, validateKubernetesExecutor(&r.Spec.ExecutorConfig, r.Namespace, w.AllowedBuildNamespaces)
}

// ValidateUpdate re-runs auth and executor validation against the updated object.
func (w *RunnerWebhook) ValidateUpdate(_ context.Context, _, newObj *Runner) (admission.Warnings, error) {
	if err := newObj.Spec.Authentication.Validate(); err != nil {
		return nil, err
	}
	if err := newObj.Spec.CACertificate.Validate(); err != nil {
		return nil, err
	}
	return nil, validateKubernetesExecutor(&newObj.Spec.ExecutorConfig, newObj.Namespace, w.AllowedBuildNamespaces)
}

// ValidateDelete is a no-op placeholder kept for future validation rules.
func (w *RunnerWebhook) ValidateDelete(_ context.Context, _ *Runner) (admission.Warnings, error) {
	return nil, nil
}
