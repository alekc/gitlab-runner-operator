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

// MultiRunnerSpec defines the desired state of MultiRunner
type MultiRunnerSpec struct {
	// +kubebuilder:validation:Minimum=1
	Concurrent int `json:"concurrent,omitempty"`

	// +kubebuilder:validation:Enum=panic;fatal;error;warning;info;debug
	LogLevel string `json:"log_level,omitempty"`

	// +kubebuilder:validation:Enum=runner;text;json
	LogFormat string `json:"log_format,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:default:3
	CheckInterval int `json:"check_interval,omitempty"`

	// SentryDsn Enables tracking of all system level errors to Sentry.
	SentryDsn string `json:"sentry_dsn,omitempty"`

	GitlabInstanceURL string `json:"gitlab_instance_url,omitempty"`

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

	// CACertificate, when set, provides a PEM CA bundle used to verify the
	// GitLab endpoint for both the operator's API calls and every runner
	// entry's own connection. Supply it inline (value) or from a Secret or
	// ConfigMap key.
	// +optional
	CACertificate *CASource `json:"caCertificate,omitempty"`

	Entries []MultiRunnerEntry `json:"entries"`
}

type MultiRunnerEntry struct {
	Name           string           `json:"name"`
	Authentication GitlabAuth       `json:"authentication"`
	ExecutorConfig KubernetesConfig `json:"executor_config,omitempty"`
	Environment    []string         `json:"environment,omitempty"`
}

// MultiRunnerStatus defines the observed state of MultiRunner. The maps are
// keyed by entry name.
type MultiRunnerStatus struct {
	Error string `json:"error,omitempty"`

	// RunnerIDs holds the GitLab numeric id per entry name for managed runners.
	RunnerIDs map[string]int `json:"runner_ids,omitempty"`

	// RegistrationHashes holds the create-options hash per entry name.
	RegistrationHashes map[string]string `json:"registration_hashes,omitempty"`

	// TokenExpiresAt holds the managed runner token expiry per entry name.
	TokenExpiresAt map[string]metav1.Time `json:"token_expires_at,omitempty"`

	// ObservedGeneration is the spec generation the controller last acted on.
	// +optional
	ObservedGeneration int64 `json:"observed_generation,omitempty"`

	// Conditions holds the latest observations of the runner state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	Ready            bool   `json:"ready"`
	ConfigMapVersion string `json:"config_map_version,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// MultiRunner is the Schema for the multirunners API
type MultiRunner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiRunnerSpec   `json:"spec,omitempty"`
	Status MultiRunnerStatus `json:"status,omitempty"`
}

func (r *MultiRunner) GetStatus() any {
	return r.Status
}

func (r *MultiRunner) finalizer() string {
	return "gitlab.k8s.alekc.dev/mr-finalizer"
}

func (r *MultiRunner) HasFinalizer() bool {
	return controllerutil.ContainsFinalizer(r, r.finalizer())
}

func (r *MultiRunner) RemoveFinalizer() {
	controllerutil.RemoveFinalizer(r, r.finalizer())
}

func (r *MultiRunner) AddFinalizer() (finalizerUpdated bool) {
	return controllerutil.AddFinalizer(r, r.finalizer())
}

func (r *MultiRunner) Update(ctx context.Context, writer client.Writer) error {
	return writer.Update(ctx, r)
}

func (r *MultiRunner) SetConfigMapVersion(versionHash string) {
	r.Status.ConfigMapVersion = versionHash
}

func (r *MultiRunner) SetStatus(newStatus any) {
	r.Status = newStatus.(MultiRunnerStatus)
}

func (r *MultiRunner) UpdateStatus(ctx context.Context, writer client.StatusWriter) error {
	return writer.Update(ctx, r)
}

func (r *MultiRunner) SetStatusReady(ready bool) {
	r.Status.Ready = ready
}

func (r *MultiRunner) ConfigMapVersion() string {
	return r.Status.ConfigMapVersion
}

// RunnerImage returns the configured gitlab-runner image, or the default.
func (r *MultiRunner) RunnerImage() string {
	if r.Spec.RunnerImage != "" {
		return r.Spec.RunnerImage
	}
	return DefaultRunnerImage
}

// CACertificate returns the custom CA source, or nil if none is configured.
func (r *MultiRunner) CACertificate() *CASource {
	return r.Spec.CACertificate
}

// SetObservedGeneration records the spec generation the controller acted on.
func (r *MultiRunner) SetObservedGeneration(gen int64) {
	r.Status.ObservedGeneration = gen
}

// SetReadyCondition updates the Ready condition on the runner status.
func (r *MultiRunner) SetReadyCondition(ready bool, reason, message string) {
	setReadyCondition(&r.Status.Conditions, r.Generation, ready, reason, message)
}

// RunnerResources returns the configured runner container resources, or a sane
// default.
func (r *MultiRunner) RunnerResources() corev1.ResourceRequirements {
	if r.Spec.RunnerResources != nil {
		return *r.Spec.RunnerResources
	}
	return defaultRunnerResources()
}

// RunnerImagePullPolicy returns the configured pull policy, or IfNotPresent.
func (r *MultiRunner) RunnerImagePullPolicy() corev1.PullPolicy {
	if r.Spec.RunnerImagePullPolicy != "" {
		return r.Spec.RunnerImagePullPolicy
	}
	return corev1.PullIfNotPresent
}

// RunnerSecurityContext returns the configured security context, or a hardened
// default.
func (r *MultiRunner) RunnerSecurityContext() *corev1.SecurityContext {
	if r.Spec.RunnerSecurityContext != nil {
		return r.Spec.RunnerSecurityContext
	}
	return defaultRunnerSecurityContext()
}

func (r *MultiRunner) RegistrationConfig() []GitlabRegInfo {
	var res []GitlabRegInfo
	for _, entry := range r.Spec.Entries {
		reg := GitlabRegInfo{
			Name:             entry.Name,
			Auth:             entry.Authentication,
			GitlabUrl:        r.Spec.GitlabInstanceURL,
			CACertificate:    r.Spec.CACertificate,
			RunnerID:         r.Status.RunnerIDs[entry.Name],
			RegistrationHash: r.Status.RegistrationHashes[entry.Name],
		}
		if exp, ok := r.Status.TokenExpiresAt[entry.Name]; ok {
			e := exp
			reg.TokenExpiresAt = &e
		}
		res = append(res, reg)
	}
	return res
}

func (r *MultiRunner) StoreRunnerRegistration(info GitlabRegInfo) {
	if r.Status.RunnerIDs == nil {
		r.Status.RunnerIDs = map[string]int{}
	}
	if r.Status.RegistrationHashes == nil {
		r.Status.RegistrationHashes = map[string]string{}
	}
	if r.Status.TokenExpiresAt == nil {
		r.Status.TokenExpiresAt = map[string]metav1.Time{}
	}
	r.Status.RunnerIDs[info.Name] = info.RunnerID
	r.Status.RegistrationHashes[info.Name] = info.RegistrationHash
	if info.TokenExpiresAt != nil {
		r.Status.TokenExpiresAt[info.Name] = *info.TokenExpiresAt
	} else {
		delete(r.Status.TokenExpiresAt, info.Name)
	}
}

func (r *MultiRunner) ChildName() string {
	return fmt.Sprintf("gitlab-mrunner-%s", r.Name)
}

// ExecutorConfigs returns the kubernetes executor config of every entry.
func (r *MultiRunner) ExecutorConfigs() []*KubernetesConfig {
	out := make([]*KubernetesConfig, 0, len(r.Spec.Entries))
	for i := range r.Spec.Entries {
		out = append(out, &r.Spec.Entries[i].ExecutorConfig)
	}
	return out
}

func (r *MultiRunner) GenerateOwnerReference() []metav1.OwnerReference {
	return []metav1.OwnerReference{{
		APIVersion:         GroupVersion.String(), // due to https://github.com/kubernetes/client-go/issues/541 type meta is empty
		Kind:               "MultiRunner",
		Name:               r.Name,
		UID:                r.UID,
		Controller:         ptr.To(true),
		BlockOwnerDeletion: nil,
	}}
}

func (r *MultiRunner) SetStatusError(errorMessage string) {
	r.Status.Error = errorMessage
}

// +kubebuilder:object:root=true

// MultiRunnerList contains a list of MultiRunner
type MultiRunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiRunner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiRunner{}, &MultiRunnerList{})
}

func (r *MultiRunner) IsBeingDeleted() bool {
	return !r.ObjectMeta.DeletionTimestamp.IsZero()
}
