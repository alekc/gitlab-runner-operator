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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RunnerSpec defines the desired state of Runner
type RunnerSpec struct {
	// Authentication configures how the runner authenticates to GitLab.
	Authentication GitlabAuth `json:"authentication"`

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

	// RunnerImage overrides the gitlab-runner container image. Defaults to
	// DefaultRunnerImage when empty.
	// +optional
	RunnerImage string `json:"runner_image,omitempty"`

	// RunnerResources overrides the resource requests/limits of the runner
	// manager container.
	// +optional
	RunnerResources *corev1.ResourceRequirements `json:"runner_resources,omitempty"`

	// RunnerImagePullPolicy overrides the runner container image pull policy.
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +optional
	RunnerImagePullPolicy corev1.PullPolicy `json:"runner_image_pull_policy,omitempty"`

	// RunnerSecurityContext overrides the runner manager container security
	// context.
	// +optional
	RunnerSecurityContext *corev1.SecurityContext `json:"runner_security_context,omitempty"`
}

// DefaultRunnerImage is the gitlab-runner image used when the spec does not
// set RunnerImage.
const DefaultRunnerImage = "gitlab/gitlab-runner:alpine-v19.1.0"

// RunnerStatus defines the observed state of Runner
type RunnerStatus struct {
	Error string `json:"error,omitempty"`

	// RunnerID is the numeric id GitLab assigned to a managed runner created
	// through the access-token path. Zero for bring-your-own-token runners.
	RunnerID int `json:"runner_id,omitempty"`

	// TokenExpiresAt is GitLab's expiry for a managed runner token, if any.
	// +optional
	TokenExpiresAt *metav1.Time `json:"token_expires_at,omitempty"`

	// RegistrationHash captures the create options that produced the current
	// managed runner; a change forces a recreate.
	RegistrationHash string `json:"registration_hash,omitempty"`

	ConfigMapVersion string `json:"config_map_version,omitempty"`

	// ObservedGeneration is the spec generation the controller last acted on.
	// +optional
	ObservedGeneration int64 `json:"observed_generation,omitempty"`

	// Conditions holds the latest observations of the runner state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready indicates that all runner operations have completed and the object
	// is ready to serve.
	Ready bool `json:"ready"`
}

// Runner is the Schema for the runners API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

func (r *Runner) RegistrationConfig() []GitlabRegInfo {
	return []GitlabRegInfo{{
		Name:             r.Name,
		Auth:             r.Spec.Authentication,
		GitlabUrl:        r.Spec.GitlabInstanceURL,
		RunnerID:         r.Status.RunnerID,
		TokenExpiresAt:   r.Status.TokenExpiresAt,
		RegistrationHash: r.Status.RegistrationHash,
	}}
}

func (r *Runner) StoreRunnerRegistration(info GitlabRegInfo) {
	r.Status.RunnerID = info.RunnerID
	r.Status.TokenExpiresAt = info.TokenExpiresAt
	r.Status.RegistrationHash = info.RegistrationHash
}

func (r *Runner) GetStatus() any {
	return r.Status
}

func (r *Runner) SetStatus(newStatus any) {
	r.Status = newStatus.(RunnerStatus)
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
		Controller:         ptr.To(true),
		BlockOwnerDeletion: nil,
	}}
}

// ChildName returns a name which should be used for all child objects generated by the runner
func (r *Runner) ChildName() string {
	return fmt.Sprintf("gitlab-runner-%s", r.Name)
}

// ExecutorConfigs returns the runner's single kubernetes executor config.
func (r *Runner) ExecutorConfigs() []*KubernetesConfig {
	return []*KubernetesConfig{&r.Spec.ExecutorConfig}
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
func (r *Runner) ConfigMapVersion() string {
	return r.Status.ConfigMapVersion
}

// RunnerImage returns the configured gitlab-runner image, or the default.
func (r *Runner) RunnerImage() string {
	if r.Spec.RunnerImage != "" {
		return r.Spec.RunnerImage
	}
	return DefaultRunnerImage
}

// SetObservedGeneration records the spec generation the controller acted on.
func (r *Runner) SetObservedGeneration(gen int64) {
	r.Status.ObservedGeneration = gen
}

// SetReadyCondition updates the Ready condition on the runner status.
func (r *Runner) SetReadyCondition(ready bool, reason, message string) {
	setReadyCondition(&r.Status.Conditions, r.Generation, ready, reason, message)
}

// RunnerResources returns the configured runner container resources, or a sane
// default.
func (r *Runner) RunnerResources() corev1.ResourceRequirements {
	if r.Spec.RunnerResources != nil {
		return *r.Spec.RunnerResources
	}
	return defaultRunnerResources()
}

// RunnerImagePullPolicy returns the configured pull policy, or IfNotPresent.
func (r *Runner) RunnerImagePullPolicy() corev1.PullPolicy {
	if r.Spec.RunnerImagePullPolicy != "" {
		return r.Spec.RunnerImagePullPolicy
	}
	return corev1.PullIfNotPresent
}

// RunnerSecurityContext returns the configured security context, or a hardened
// default.
func (r *Runner) RunnerSecurityContext() *corev1.SecurityContext {
	if r.Spec.RunnerSecurityContext != nil {
		return r.Spec.RunnerSecurityContext
	}
	return defaultRunnerSecurityContext()
}
