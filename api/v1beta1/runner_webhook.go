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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var runnerlog = logf.Log.WithName("runner-resource")

func (r *Runner) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-gitlab-k8s-alekc-dev-v1beta1-runner,mutating=true,failurePolicy=fail,sideEffects=None,groups=gitlab.k8s.alekc.dev,resources=runners,verbs=create;update,versions=v1beta1,name=mrunner.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Runner{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Runner) Default() {
	if r.Spec.GitlabInstanceURL == "" {
		r.Spec.GitlabInstanceURL = "https://gitlab.com/"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-gitlab-k8s-alekc-dev-v1beta1-runner,mutating=false,failurePolicy=fail,sideEffects=None,groups=gitlab.k8s.alekc.dev,resources=runners,verbs=create;update,versions=v1beta1,name=vrunner.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Runner{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Runner) ValidateCreate() error {
	// runnerlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Runner) ValidateUpdate(old runtime.Object) error {
	// runnerlog.Info("validate update", "name", r.Name)

	// we do not want to permit changing of the name of the runner (or better, we'd rather avoid dealing with cleaning)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Runner) ValidateDelete() error {
	// runnerlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
