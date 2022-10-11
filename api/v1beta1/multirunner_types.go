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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// Name The runner’s description. Informational only.
	Name               string
	RegistrationConfig RegisterNewRunnerOptions `json:"registration_config"`
	ExecutorConfig     KubernetesConfig         `json:"executor_config,omitempty"`
	Environment        []string                 `json:"environment,omitempty"`
}

// MultiRunnerStatus defines the observed state of MultiRunner
type MultiRunnerStatus struct {
	Error                string            `json:"error"`
	AuthTokens           map[string]string `json:"auth_tokens"`
	LastRegistrationTags map[string]string `json:"last_registration_tags"`
	Ready                bool              `json:"ready"`
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

func (in *MultiRunner) ChildName() string {
	// TODO implement me
	panic("implement me")
}

func (in *MultiRunner) GenerateOwnerReference() []metav1.OwnerReference {
	// TODO implement me
	panic("implement me")
}

func (in *MultiRunner) SetStatusError(errorMessage string) {
	// TODO implement me
	panic("implement me")
}

func (in *MultiRunner) SetStatusConfigMapVersion(versionHash string) {
	// TODO implement me
	panic("implement me")
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

func (in *MultiRunner) IsBeingDeleted() bool {
	return !in.ObjectMeta.DeletionTimestamp.IsZero()
}
