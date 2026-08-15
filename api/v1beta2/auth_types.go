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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
)

// SecretKeySelector points at a single key inside a Secret in the runner's
// namespace. It mirrors corev1.SecretKeySelector but makes Key optional so it
// can default to "token"; the upstream type marks Key required, which would
// force every reference to spell it out.
type SecretKeySelector struct {
	// Name of the Secret in the runner's namespace.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Key holding the token. Defaults to "token" when omitted.
	// +optional
	Key string `json:"key,omitempty"`

	// Optional, when true, lets a missing secret or key resolve to an empty
	// token instead of failing.
	// +optional
	Optional *bool `json:"optional,omitempty"`
}

// TokenSource supplies a credential in one of two mutually exclusive ways: an
// inline literal value, or a reference to a key inside a Kubernetes Secret in
// the runner's namespace. Exactly one of Value / SecretKeyRef may be set.
//
// +kubebuilder:validation:XValidation:rule="has(self.value) != has(self.secret_key_ref)",message="set either value or secret_key_ref, not both"
type TokenSource struct {
	// Value is the literal token. Convenient for testing; prefer SecretKeyRef
	// in production so the token is not stored in the object spec.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretKeyRef reads the token from a Secret in the runner's namespace. The
	// referenced key defaults to "token" when Key is omitted. Optional is
	// honoured: when true, a missing secret or key resolves to an empty token
	// instead of failing.
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secret_key_ref,omitempty"`
}

// IsSet reports whether the source supplies a token through either mode. It is
// nil-safe so callers can probe an unset (*TokenSource) field directly.
func (t *TokenSource) IsSet() bool {
	return t != nil && (t.Value != "" || t.SecretKeyRef != nil)
}

// GitlabAuth configures how a runner authenticates to GitLab. GitLab removed
// the legacy registration-token workflow (deprecated in 16.0, disabled by
// default from 18.0); runners now authenticate with a runner authentication
// token (the "glrt-" token). Exactly one of two modes must be provided:
//
//   - Bring-your-own token: set Token to a runner authentication token created
//     in the GitLab UI or via the API. The operator performs no GitLab API
//     calls and writes the token straight into the runner config.
//
//   - Managed: set AccessToken to a personal, group, or project access token
//     holding the "create_runner" scope, together with a CreateOptions block.
//     The operator creates the runner through POST /user/runners, stores the
//     returned token, and deletes the runner from GitLab when the object is
//     removed.
//
// Each credential is a TokenSource, so it may be supplied inline (value) or
// from a Secret key (secret_key_ref, with a configurable key defaulting to
// "token").
//
// +kubebuilder:validation:XValidation:rule="has(self.token) || has(self.create_options)",message="one of token or create_options must be set"
// +kubebuilder:validation:XValidation:rule="!(has(self.token) && has(self.create_options))",message="set either a pre-created authentication token or create_options, not both"
// +kubebuilder:validation:XValidation:rule="!has(self.create_options) || has(self.access_token)",message="create_options requires access_token"
// +kubebuilder:validation:XValidation:rule="!has(self.access_token) || has(self.create_options)",message="access_token is only used with create_options (managed mode)"
type GitlabAuth struct {
	// Token is the pre-created runner authentication token ("glrt-...") used in
	// bring-your-own mode. Mutually exclusive with the managed CreateOptions.
	// +optional
	Token *TokenSource `json:"token,omitempty"`

	// AccessToken is a personal, group, or project access token with the
	// "create_runner" scope. Required for the managed mode.
	// +optional
	AccessToken *TokenSource `json:"access_token,omitempty"`

	// CreateOptions describes the runner to create. When set, the operator runs
	// in managed mode and owns the runner's lifecycle on GitLab.
	// +optional
	CreateOptions *RunnerCreateOptions `json:"create_options,omitempty"`
}

// IsManaged reports whether the operator should create and own the runner on
// GitLab (managed mode) rather than consume a pre-created token.
func (a *GitlabAuth) IsManaged() bool {
	return a != nil && a.CreateOptions != nil
}

// RunnerCreateOptions mirrors the POST /user/runners request body. It is only
// used in managed mode.
//
// +kubebuilder:validation:XValidation:rule="self.runner_type != 'group_type' || has(self.group_id)",message="group_type runner requires group_id"
// +kubebuilder:validation:XValidation:rule="self.runner_type != 'project_type' || has(self.project_id)",message="project_type runner requires project_id"
type RunnerCreateOptions struct {
	// RunnerType selects the scope of the runner to create.
	// +kubebuilder:validation:Enum=instance_type;group_type;project_type
	RunnerType string `json:"runner_type"`

	// GroupID is required when RunnerType is group_type.
	// +optional
	GroupID *int `json:"group_id,omitempty"`

	// ProjectID is required when RunnerType is project_type.
	// +optional
	ProjectID *int `json:"project_id,omitempty"`

	// +optional
	Description string `json:"description,omitempty"`

	// +optional
	Paused *bool `json:"paused,omitempty"`

	// +optional
	Locked *bool `json:"locked,omitempty"`

	// +optional
	RunUntagged *bool `json:"run_untagged,omitempty"`

	// +optional
	TagList []string `json:"tag_list,omitempty"`

	// +optional
	AccessLevel string `json:"access_level,omitempty"`

	// +optional
	MaximumTimeout *int `json:"maximum_timeout,omitempty"`

	// +optional
	MaintenanceNote string `json:"maintenance_note,omitempty"`
}

// Hash returns a stable digest of the create options. The operator stores it in
// status and recreates the runner whenever the desired options change.
func (o *RunnerCreateOptions) Hash() string {
	if o == nil {
		return ""
	}
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	}
	sum := sha1.Sum(b)
	return hex.EncodeToString(sum[:])
}
