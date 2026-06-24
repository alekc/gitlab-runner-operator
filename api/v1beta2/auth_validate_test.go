package v1beta2

import "testing"

// TestGitlabAuthValidate exercises the mode-exclusivity and per-source rules
// that the admission webhook relies on.
func TestGitlabAuthValidate(t *testing.T) {
	cases := []struct {
		name    string
		auth    GitlabAuth
		wantErr bool
	}{
		{
			name: "byo inline ok",
			auth: GitlabAuth{Token: &TokenSource{Value: "glrt-x"}},
		},
		{
			name: "byo secret ok",
			auth: GitlabAuth{Token: &TokenSource{SecretKeyRef: &SecretKeySelector{Name: "s"}}},
		},
		{
			name:    "byo value and secret both set",
			auth:    GitlabAuth{Token: &TokenSource{Value: "x", SecretKeyRef: &SecretKeySelector{Name: "s"}}},
			wantErr: true,
		},
		{
			name:    "secret ref missing name",
			auth:    GitlabAuth{Token: &TokenSource{SecretKeyRef: &SecretKeySelector{}}},
			wantErr: true,
		},
		{
			name:    "nothing set",
			auth:    GitlabAuth{},
			wantErr: true,
		},
		{
			name:    "byo and managed both",
			auth:    GitlabAuth{Token: &TokenSource{Value: "x"}, CreateOptions: &RunnerCreateOptions{RunnerType: "instance_type"}},
			wantErr: true,
		},
		{
			name:    "managed without access token",
			auth:    GitlabAuth{CreateOptions: &RunnerCreateOptions{RunnerType: "instance_type"}},
			wantErr: true,
		},
		{
			name: "managed ok",
			auth: GitlabAuth{AccessToken: &TokenSource{Value: "glpat-x"}, CreateOptions: &RunnerCreateOptions{RunnerType: "instance_type"}},
		},
		{
			name:    "group type without group id",
			auth:    GitlabAuth{AccessToken: &TokenSource{Value: "x"}, CreateOptions: &RunnerCreateOptions{RunnerType: "group_type"}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.auth.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
