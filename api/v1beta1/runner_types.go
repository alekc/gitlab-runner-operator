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
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RunnerSpec defines the desired state of Runner
type RunnerSpec struct {
	RegistrationConfig RegisterNewRunnerOptions `json:"registration_config"`

	// +kubebuilder:validation:Optional
	GitlabInstanceURL string `json:"gitlab_instance_url,omitempty"`

	// +kubebuilder:validation:Enum=panic;fatal;error;warning;info;debug
	LogLevel string `json:"log_level,omitempty"`

	// +kubebuilder:validation:Minimum=1
	Concurrent int `json:"concurrent,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:default:3
	CheckInterval int `json:"check_interval,omitempty"`

	// +kubebuilder:validation:Enum=runner;text;json
	LogFormat string `json:"log_format,omitempty"`

	ExecutorConfig KubernetesConfig `json:"executor_config,omitempty"`

	// +kubebuilder:validation:Optional
	// Environment contains custom environment variables injected to build environment
	Environment []string `json:"environment,omitempty"`
}

// RunnerStatus defines the observed state of Runner
type RunnerStatus struct {
	Error string `json:"error"`

	// LastRegistrationToken is the last token used for a successful authentication
	LastRegistrationToken string `json:"last_registration_token"`

	// LastRegistrationTags are last tags used in successful registration
	LastRegistrationTags []string `json:"last_registration_tags"`

	// AuthenticationToken obtained from the gitlab which can be used in runner configuration for authentication
	AuthenticationToken string `json:"authentication_token"`
	ConfigMapVersion    string `json:"config_map_version"`

	// Ready indicates that all runner operation has been completed and final object is ready to serve
	Ready bool `json:"ready"`
}

// Runner is the Schema for the runners API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

func (in *Runner) RegistrationConfig() []GitlabRegInfo {
	return []GitlabRegInfo{{
		RegisterNewRunnerOptions: in.Spec.RegistrationConfig,
		AuthToken:                in.Status.AuthenticationToken,
		GitlabUrl:                in.Spec.GitlabInstanceURL,
	}}
}

func (in *Runner) StoreRunnerRegistration(info GitlabRegInfo) {
	in.Status.AuthenticationToken = info.AuthToken
	in.Status.LastRegistrationToken = *info.Token
	in.Status.LastRegistrationTags = info.TagList
}

func (in *Runner) GitlabRegTokens() []string {
	return []string{in.Status.AuthenticationToken}
}

func (in *Runner) GetStatus() any {
	return in.Status
}

func (in *Runner) SetStatus(newStatus any) {
	in.Status = newStatus.(RunnerStatus)
}

// +kubebuilder:object:root=true

// RunnerList contains a list of Runner
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Runner{}, &RunnerList{})
}

// GetAnnotation returns the annotation value for a given key.
// If the key doesn't exist, an empty string is returned
func (r *Runner) GetAnnotation(key string) string {
	annotations := r.GetAnnotations()
	if annotations == nil {
		return ""
	}
	if val, ok := annotations[key]; ok {
		return val
	}
	return ""
}

// GenerateOwnerReference returns a generated owner reference which can be used later to for any child object
func (r *Runner) GenerateOwnerReference() []metav1.OwnerReference {
	return []metav1.OwnerReference{{
		APIVersion:         GroupVersion.String(), // due to https://github.com/kubernetes/client-go/issues/541 type meta is empty
		Kind:               "Runner",
		Name:               r.Name,
		UID:                r.UID,
		Controller:         pointer.BoolPtr(true),
		BlockOwnerDeletion: nil,
	}}
}

// ChildName returns a name which should be used for all child objects generated by the runner
func (r *Runner) ChildName() string {
	return fmt.Sprintf("gitlab-runner-%s", r.Name)
}

// Namespace returns namespace of the runner
func (r *Runner) GetNamespace() string {
	return r.Namespace
}

// SetStatusError sets the error on the status
func (r *Runner) SetStatusError(errorMessage string) {
	r.Status.Error = errorMessage
}

// SetStatusConfigMapVersion sets configm map version hash in the runner status
func (r *Runner) SetConfigMapVersion(versionHash string) {
	r.Status.ConfigMapVersion = versionHash
}

func (r *Runner) IsBeingDeleted() bool {
	return !r.ObjectMeta.DeletionTimestamp.IsZero()
}

func (r *Runner) IsAuthenticated() bool {
	return r.Status.AuthenticationToken != ""
}
func (r *Runner) RemoveFinalizer() {
	controllerutil.RemoveFinalizer(r, r.finalizer())
}
func (r *Runner) Update(ctx context.Context, writer client.Writer) error {
	return writer.Update(ctx, r)
}
func (r *Runner) AddFinalizer() bool {
	return controllerutil.AddFinalizer(r, r.finalizer())
}

func (r *Runner) UpdateStatus(ctx context.Context, writer client.StatusWriter) error {
	return writer.Update(ctx, r)
}
func (r *Runner) finalizer() string {
	return "gitlab.k8s.alekc.dev/finalizer"
}
func (r *Runner) HasFinalizer() bool {
	return controllerutil.ContainsFinalizer(r, r.finalizer())
}
func (r *Runner) SetStatusReady(ready bool) {
	r.Status.Ready = ready
}
func (r *Runner) HasValidAuth() bool {
	return !(r.Status.AuthenticationToken == "" ||
		r.Status.LastRegistrationToken != *r.Spec.RegistrationConfig.Token ||
		!reflect.DeepEqual(r.Status.LastRegistrationTags, r.Spec.RegistrationConfig.TagList))
}

// func (r *Runner) RegisterOnGitlab(gitlabClient api2.GitlabClient, logger logr.Logger) (ctrl.Result, error) {
// 	// since we are doing a new registration, IF the runner already has an authentication token, delete it from gitlab server
// 	if r.Status.AuthenticationToken != "" {
// 		_, err := gitlabClient.DeleteByToken(r.Status.AuthenticationToken)
// 		if err != nil {
// 			logger.Error(err, "cannot remove gitlab runner registration")
// 			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 30}, err
// 		}
// 	}
// 	token, err := gitlabClient.Register(r.Spec.RegistrationConfig)
// 	if err != nil {
// 		logger.Error(err, "cannot register runner on the gitlab")
// 		r.Status.Error = fmt.Sprintf("Cannot register the runner on gitlab api. %s", err.Error())
// 		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
// 	}
// 	r.Status.AuthenticationToken = token
// 	r.Status.LastRegistrationToken = *r.Spec.RegistrationConfig.Token
// 	r.Status.LastRegistrationTags = r.Spec.RegistrationConfig.TagList
//
// 	logger.Info("registered a new runner on gitlab")
// 	return ctrl.Result{Requeue: true}, nil
// }

// func (r *Runner) DeleteFromGitlab(apiClient api2.GitlabClient, logger logr.Logger) error {
// 	// _, err := apiClient.DeleteByToken(r.Status.AuthenticationToken)
// 	// if err != nil {
// 	// 	logger.Error(err, "cannot remove gitlab runner from the gitlab server")
// 	// }
// 	// return err
// 	return nil
// }

func (r *Runner) ConfigMapVersion() string {
	return r.Status.ConfigMapVersion
}
