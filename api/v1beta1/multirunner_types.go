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

	Entries []MultiRunnerEntry `json:"entries"`
}

type MultiRunnerEntry struct {
	Name               string                   `json:"name"`
	RegistrationConfig RegisterNewRunnerOptions `json:"registration_config"`
	ExecutorConfig     KubernetesConfig         `json:"executor_config,omitempty"`
	Environment        []string                 `json:"environment,omitempty"`
}

// MultiRunnerStatus defines the observed state of MultiRunner
type MultiRunnerStatus struct {
	Error                string              `json:"error"`
	AuthTokens           map[string]string   `json:"auth_tokens"`
	LastRegistrationTags map[string][]string `json:"last_registration_tags"`
	Ready                bool                `json:"ready"`
	ConfigMapVersion     string              `json:"config_map_version"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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

func (r *MultiRunner) IsAuthenticated() bool {
	// every entry r our runner book must have its auth
	for _, entry := range r.Spec.Entries {
		if _, found := r.Status.AuthTokens[*entry.RegistrationConfig.Token]; !found {
			return false
		}
	}
	return true
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

// HasValidAuth verify if one or more entries from registration has to be registered again
func (r *MultiRunner) HasValidAuth() bool {
	// every entry r our runner book must have its auth
	for _, entry := range r.Spec.Entries {
		// do we have a valid token?
		token := *entry.RegistrationConfig.Token
		if _, found := r.Status.AuthTokens[token]; !found {
			return false
		}
		// old tags must be stored and they need to match the present one
		if oldTags, found := r.Status.LastRegistrationTags[token]; !found || !reflect.DeepEqual(oldTags, entry.RegistrationConfig.TagList) {
			return false
		}
	}
	return true
}

func (r *MultiRunner) ConfigMapVersion() string {
	return r.Status.ConfigMapVersion
}

func (r *MultiRunner) RegistrationConfig() []GitlabRegInfo {
	var res []GitlabRegInfo
	for _, entry := range r.Spec.Entries {
		token := *entry.RegistrationConfig.Token
		res = append(res, GitlabRegInfo{
			RegisterNewRunnerOptions: entry.RegistrationConfig,
			AuthToken:                r.Status.AuthTokens[token],
			GitlabUrl:                r.Spec.GitlabInstanceURL,
		})
	}
	return res
}

func (r *MultiRunner) StoreRunnerRegistration(info GitlabRegInfo) {
	token := *info.RegisterNewRunnerOptions.Token
	r.Status.AuthTokens[token] = info.AuthToken
	r.Status.LastRegistrationTags[token] = info.TagList
}

func (r *MultiRunner) ChildName() string {
	return fmt.Sprintf("gitlab-mrunner-%s", r.Name)
}

func (r *MultiRunner) GenerateOwnerReference() []metav1.OwnerReference {
	return []metav1.OwnerReference{{
		APIVersion:         GroupVersion.String(), // due to https://github.com/kubernetes/client-go/issues/541 type meta is empty
		Kind:               "MultirRunner",
		Name:               r.Name,
		UID:                r.UID,
		Controller:         pointer.Bool(true),
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
