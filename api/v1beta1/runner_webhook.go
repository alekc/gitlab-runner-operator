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

package v1beta1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager registers the defaulting and validating webhooks for
// the Runner type with the manager.
func (r *Runner) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(&RunnerWebhook{}).
		WithValidator(&RunnerWebhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-gitlab-k8s-alekc-dev-v1beta1-runner,mutating=true,failurePolicy=fail,sideEffects=None,groups=gitlab.k8s.alekc.dev,resources=runners,verbs=create;update,versions=v1beta1,name=mrunner.kb.io,admissionReviewVersions={v1,v1beta1}
// +kubebuilder:webhook:path=/validate-gitlab-k8s-alekc-dev-v1beta1-runner,mutating=false,failurePolicy=fail,sideEffects=None,groups=gitlab.k8s.alekc.dev,resources=runners,verbs=create;update,versions=v1beta1,name=vrunner.kb.io,admissionReviewVersions={v1,v1beta1}

// RunnerWebhook implements the controller-runtime defaulting and validating
// webhook interfaces for the Runner type. controller-runtime 0.19+ moved these
// off the API type onto a dedicated handler implementing the generic
// Defaulter / Validator interfaces.
type RunnerWebhook struct{}

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

// ValidateCreate is a no-op placeholder kept for future validation rules.
func (w *RunnerWebhook) ValidateCreate(_ context.Context, _ *Runner) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate is a no-op placeholder kept for future validation rules.
func (w *RunnerWebhook) ValidateUpdate(_ context.Context, _, _ *Runner) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete is a no-op placeholder kept for future validation rules.
func (w *RunnerWebhook) ValidateDelete(_ context.Context, _ *Runner) (admission.Warnings, error) {
	return nil, nil
}
