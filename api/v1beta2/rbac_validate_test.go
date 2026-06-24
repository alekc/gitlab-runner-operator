package v1beta2

import "testing"

// TestValidateKubernetesExecutor checks that dynamic-namespace executor
// settings are rejected, because the operator pre-provisions namespaced RBAC.
func TestValidateKubernetesExecutor(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *KubernetesConfig
		wantErr bool
	}{
		{name: "nil ok", cfg: nil},
		{name: "empty ok", cfg: &KubernetesConfig{}},
		{name: "static namespace ok", cfg: &KubernetesConfig{Namespace: "build"}},
		{name: "namespace_per_job rejected", cfg: &KubernetesConfig{NamespacePerJob: true}, wantErr: true},
		{name: "namespace_overwrite_allowed rejected", cfg: &KubernetesConfig{NamespaceOverwriteAllowed: ".*"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateKubernetesExecutor(tc.cfg)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
