package v1beta2

import "testing"

// TestValidateKubernetesExecutor checks that dynamic-namespace executor settings
// are rejected, and that a static build namespace is confined to the runner's
// own namespace unless an operator allowlist opts others in.
func TestValidateKubernetesExecutor(t *testing.T) {
	const own = "runner-ns"
	cases := []struct {
		name    string
		cfg     *KubernetesConfig
		allowed []string
		wantErr bool
	}{
		{name: "nil ok", cfg: nil},
		{name: "empty namespace ok", cfg: &KubernetesConfig{}},
		{name: "own namespace ok", cfg: &KubernetesConfig{Namespace: own}},
		{name: "cross namespace rejected by default", cfg: &KubernetesConfig{Namespace: "build"}, wantErr: true},
		{name: "cross namespace allowed when listed", cfg: &KubernetesConfig{Namespace: "build"}, allowed: []string{"build"}},
		{name: "cross namespace allowed via wildcard", cfg: &KubernetesConfig{Namespace: "kube-system"}, allowed: []string{"*"}},
		{name: "cross namespace not in list rejected", cfg: &KubernetesConfig{Namespace: "kube-system"}, allowed: []string{"build"}, wantErr: true},
		{name: "namespace_per_job rejected", cfg: &KubernetesConfig{NamespacePerJob: true}, wantErr: true},
		{name: "namespace_overwrite_allowed rejected", cfg: &KubernetesConfig{NamespaceOverwriteAllowed: ".*"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateKubernetesExecutor(tc.cfg, own, tc.allowed)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
