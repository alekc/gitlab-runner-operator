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
	"fmt"
)

// GitlabAuth configures how a runner authenticates to GitLab. GitLab removed
// the legacy registration-token workflow (deprecated in 16.0, disabled by
// default from 18.0); runners now authenticate with a runner authentication
// token (the "glrt-" token). Exactly one of two modes must be provided:
//
//   - Bring-your-own token: set AuthenticationToken (or
//     AuthenticationTokenSecret) to a runner authentication token created in
//     the GitLab UI or via the API. The operator performs no GitLab API calls
//     and writes the token straight into the runner config.
//
//   - Managed: set AccessToken (or AccessTokenSecret) to a personal, group, or
//     project access token holding the "create_runner" scope, together with a
//     CreateOptions block. The operator creates the runner through
//     POST /user/runners, stores the returned token, and deletes the runner
//     from GitLab when the object is removed.
type GitlabAuth struct {
	// AuthenticationToken is a pre-created runner authentication token
	// ("glrt-..."). Mutually exclusive with the managed (CreateOptions) mode.
	// +optional
	AuthenticationToken string `json:"authentication_token,omitempty"`

	// AuthenticationTokenSecret names a secret in the runner's namespace whose
	// "token" key holds the runner authentication token.
	// +optional
	AuthenticationTokenSecret string `json:"authentication_token_secret,omitempty"`

	// AccessToken is a personal, group, or project access token with the
	// "create_runner" scope. Required for the managed mode.
	// +optional
	AccessToken string `json:"access_token,omitempty"`

	// AccessTokenSecret names a secret in the runner's namespace whose "token"
	// key holds the access token used for the managed mode.
	// +optional
	AccessTokenSecret string `json:"access_token_secret,omitempty"`

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

// Validate checks that exactly one auth mode is configured and that managed
// mode has the inputs it needs. It is called from the admission webhook.
func (a GitlabAuth) Validate() error {
	hasByo := a.AuthenticationToken != "" || a.AuthenticationTokenSecret != ""
	hasManaged := a.CreateOptions != nil
	switch {
	case hasByo && hasManaged:
		return fmt.Errorf("set either a pre-created authentication token or create_options, not both")
	case !hasByo && !hasManaged:
		return fmt.Errorf("one of authentication_token, authentication_token_secret, or create_options must be set")
	case hasManaged && a.AccessToken == "" && a.AccessTokenSecret == "":
		return fmt.Errorf("create_options requires access_token or access_token_secret")
	}
	if a.CreateOptions != nil {
		switch a.CreateOptions.RunnerType {
		case "group_type":
			if a.CreateOptions.GroupID == nil {
				return fmt.Errorf("group_type runner requires group_id")
			}
		case "project_type":
			if a.CreateOptions.ProjectID == nil {
				return fmt.Errorf("project_type runner requires project_id")
			}
		}
	}
	return nil
}

// RunnerCreateOptions mirrors the POST /user/runners request body. It is only
// used in managed mode.
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
