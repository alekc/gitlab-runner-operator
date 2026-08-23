package generate

import (
	"strings"
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func wantTLSCALine() string {
	return `tls-ca-file = "` + types.CACertFile + `"`
}

func TestSingleRunnerConfig_TLSCAFile(t *testing.T) {
	base := &v1beta2.Runner{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns"},
		Spec:       v1beta2.RunnerSpec{GitlabInstanceURL: "https://gitlab.example.com"},
	}
	tokens := map[string]string{"r1": "glrt-token"}

	cfg, _, err := SingleRunnerConfig(base, tokens, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(cfg, "tls-ca-file") {
		t.Fatalf("did not expect tls-ca-file without a CA, got:\n%s", cfg)
	}

	withCA := base.DeepCopy()
	withCA.Spec.CACertificate = &v1beta2.CASource{ConfigMapKeyRef: &v1beta2.CAKeyRef{Name: "ca-cm"}}
	cfg, _, err = SingleRunnerConfig(withCA, tokens, []byte("ca-pem"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cfg, wantTLSCALine()) {
		t.Fatalf("expected %q in config, got:\n%s", wantTLSCALine(), cfg)
	}

	// an inline value sets tls-ca-file the same way as a ref
	inline := base.DeepCopy()
	inline.Spec.CACertificate = &v1beta2.CASource{Value: "-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----\n"}
	cfg, _, err = SingleRunnerConfig(inline, tokens, []byte("ca-pem"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cfg, wantTLSCALine()) {
		t.Fatalf("expected %q for an inline CA, got:\n%s", wantTLSCALine(), cfg)
	}
}

func TestMultiRunnerConfig_TLSCAFile(t *testing.T) {
	base := &v1beta2.MultiRunner{
		ObjectMeta: metav1.ObjectMeta{Name: "m1", Namespace: "ns"},
		Spec: v1beta2.MultiRunnerSpec{
			GitlabInstanceURL: "https://gitlab.example.com",
			Entries:           []v1beta2.MultiRunnerEntry{{Name: "e1"}, {Name: "e2"}},
		},
	}
	tokens := map[string]string{"e1": "glrt-a", "e2": "glrt-b"}

	cfg, _, err := MultiRunnerConfig(base, tokens, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(cfg, "tls-ca-file") {
		t.Fatalf("did not expect tls-ca-file without a CA, got:\n%s", cfg)
	}

	withCA := base.DeepCopy()
	withCA.Spec.CACertificate = &v1beta2.CASource{SecretKeyRef: &v1beta2.CAKeyRef{Name: "ca-secret"}}
	cfg, _, err = MultiRunnerConfig(withCA, tokens, []byte("ca-pem"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// every entry must carry the CA file
	if got := strings.Count(cfg, wantTLSCALine()); got != 2 {
		t.Fatalf("expected tls-ca-file on both entries (2), got %d in:\n%s", got, cfg)
	}
}

// TestSingleRunnerConfig_CAInHash verifies the CA bytes are folded into the
// config hash, so a CA rotation rolls the deployment even though config.toml
// only references tls-ca-file by path.
func TestSingleRunnerConfig_CAInHash(t *testing.T) {
	r := &v1beta2.Runner{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns"},
		Spec: v1beta2.RunnerSpec{
			GitlabInstanceURL: "https://gitlab.example.com",
			CACertificate:     &v1beta2.CASource{Value: "inline"},
		},
	}
	tokens := map[string]string{"r1": "glrt-token"}

	_, h1, err := SingleRunnerConfig(r, tokens, []byte("ca-one"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, h2, err := SingleRunnerConfig(r, tokens, []byte("ca-two"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 == h2 {
		t.Fatal("expected the config hash to change when the CA bundle changes")
	}
	_, h3, err := SingleRunnerConfig(r, tokens, []byte("ca-one"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 != h3 {
		t.Fatal("expected the same CA bundle to produce the same hash")
	}
}

// Every executor key added for #58, set on one Runner and asserted in the
// rendered config.toml. Asserting on the rendered output rather than on struct
// tags is what proves the key actually reaches the runner.
func TestSingleRunnerConfig_ExposedExecutorKeys(t *testing.T) {
	truthy := true
	limit := 7
	backoff := 2000

	r := &v1beta2.Runner{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns"},
		Spec: v1beta2.RunnerSpec{
			GitlabInstanceURL: "https://gitlab.example.com",
			ExecutorConfig: v1beta2.KubernetesConfig{
				AllowedUsers:                      []string{"1000"},
				AllowedGroups:                     []string{"2000"},
				AutomountServiceAccountToken:      &truthy,
				PodDisruptionBudget:               &truthy,
				PrintPodWarningEvents:             &truthy,
				UseServiceAccountImagePullSecrets: true,
				CleanupResourcesTimeout:           "5m",
				RequestRetryLimit:                 &limit,
				RequestRetryLimits:                map[string]int{"connection refused": 3},
				RequestRetryBackoffMax:            &backoff,
				Services:                          []v1beta2.Service{{Name: "postgres:16", Environment: []string{"POSTGRES_DB=test"}}},
				PodSecurityContext: &v1beta2.KubernetesPodSecurityContext{
					SELinuxType:     "spc_t",
					AppArmorProfile: &v1beta2.KubernetesAppArmorProfile{Type: "Localhost", LocalhostProfile: "k8s-apparmor-example-deny-write"},
					SeccompProfile:  &v1beta2.KubernetesSeccompProfile{Type: "Localhost", LocalhostProfile: "profiles/audit.json"},
				},
				BuildContainerSecurityContext: &v1beta2.KubernetesContainerSecurityContext{ProcMount: "Unmasked"},
				Affinity: &v1beta2.KubernetesAffinity{
					PodAffinity: &v1beta2.KubernetesPodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []v1beta2.PodAffinityTerm{{
							TopologyKey:       "kubernetes.io/hostname",
							MatchLabelKeys:    []string{"pod-template-hash"},
							MismatchLabelKeys: []string{"release"},
						}},
					},
				},
				HelperContainerSecurityContext: &v1beta2.KubernetesContainerSecurityContext{
					SELinuxType:     "spc_t",
					SeccompProfile:  &v1beta2.KubernetesSeccompProfile{Type: "RuntimeDefault"},
					AppArmorProfile: &v1beta2.KubernetesAppArmorProfile{Type: "RuntimeDefault"},
				},
				Volumes: &v1beta2.KubernetesVolumes{
					HostPaths: []v1beta2.KubernetesHostPath{{
						Name: "hp", MountPath: "/mnt/hp", HostPath: "/srv/hp",
						MountPropagation: &[]string{"HostToContainer"}[0],
					}},
					EmptyDirs: []v1beta2.KubernetesEmptyDir{{Name: "ed", MountPath: "/mnt/ed", SizeLimit: "1Gi",
						MountPropagation: &[]string{"None"}[0]}},
					PVCs: []v1beta2.KubernetesPVC{{Name: "pvc", MountPath: "/mnt/pvc",
						MountPropagation: &[]string{"Bidirectional"}[0]}},
					NFSVolumes: []v1beta2.KubernetesNFS{{Name: "nfs", MountPath: "/mnt/nfs", Server: "10.0.0.1", Path: "/exports"}},
				},
			},
		},
	}

	cfg, _, err := SingleRunnerConfig(r, map[string]string{"r1": "glrt-token"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Values, not just key names: a key rendered with the wrong value is worse
	// than a missing one, because it looks configured.
	for _, want := range []string{
		`allowed_users = ["1000"]`,
		`allowed_groups = ["2000"]`,
		`automount_service_account_token = true`,
		`pod_disruption_budget = true`,
		`print_pod_warning_events = true`,
		`use_service_account_image_pull_secrets = true`,
		`cleanup_resources_timeout = "5m"`,
		`retry_limit = 7`,
		`retry_backoff_max = 2000`,
		`[runners.kubernetes.retry_limits]`,
		`"connection refused" = 3`,
		`match_label_keys = ["pod-template-hash"]`,
		`mismatch_label_keys = ["release"]`,
		`[runners.kubernetes.helper_container_security_context]`,
		`[runners.kubernetes.helper_container_security_context.seccomp_profile]`,
		`[runners.kubernetes.helper_container_security_context.app_armor_profile]`,
		`mount_propagation = "None"`,
		`mount_propagation = "Bidirectional"`,
		`environment = ["POSTGRES_DB=test"]`,
		`selinux_type = "spc_t"`,
		`[runners.kubernetes.pod_security_context.app_armor_profile]`,
		`localhost_profile = "k8s-apparmor-example-deny-write"`,
		`[runners.kubernetes.pod_security_context.seccomp_profile]`,
		`localhost_profile = "profiles/audit.json"`,
		`proc_mount = "Unmasked"`,
		`mount_propagation = "HostToContainer"`,
		`size_limit = "1Gi"`,
		`[[runners.kubernetes.volumes.nfs]]`,
		`server = "10.0.0.1"`,
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("missing %q in rendered config:\n%s", want, cfg)
		}
	}
}

// The CRD defaults these, and the generator floors them again because helm does
// not update CRDs on upgrade. Both halves are asserted: explicit values render,
// and a zero renders the default rather than dropping the key.
func TestSingleRunnerConfig_ConcurrencyKeys(t *testing.T) {
	base := &v1beta2.Runner{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns"},
		Spec: v1beta2.RunnerSpec{
			GitlabInstanceURL: "https://gitlab.example.com",
			Concurrent:        50,
			ConcurrencyLimits: v1beta2.ConcurrencyLimits{Limit: 50, RequestConcurrency: 7},
		},
	}
	tokens := map[string]string{"r1": "glrt-token"}

	cfg, _, err := SingleRunnerConfig(base, tokens, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Anchored on the two-space indent inside [[runners]]: "limit = 5" is a
	// prefix of "limit = 50", and the executor block has a dozen keys ending
	// in "limit" at a deeper indent.
	for _, want := range []string{"\n  limit = 50", "\n  request_concurrency = 7", "\nconcurrent = 50"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("expected %q in config, got:\n%s", want, cfg)
		}
	}

	// An object the apiserver never defaulted, which is what a helm upgrade
	// leaves behind when the CRDs are not re-applied. Without the code-side
	// floor the keys vanish and gitlab-runner treats limit as unlimited.
	bare := base.DeepCopy()
	bare.Spec.Limit = 0
	bare.Spec.RequestConcurrency = 0
	cfg, _, err = SingleRunnerConfig(bare, tokens, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"\n  limit = 10", "\n  request_concurrency = 3"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("expected the floored %q, got:\n%s", want, cfg)
		}
	}
}

// runnerBlock returns the [[runners]] block for the named runner, so a value
// assertion belongs to one entry rather than to the whole file.
func runnerBlock(t *testing.T, cfg, name string) string {
	t.Helper()
	for _, block := range strings.Split(cfg, "[[runners]]") {
		if strings.Contains(block, `name = "`+name+`"`) {
			return block
		}
	}
	t.Fatalf("no [[runners]] block named %q in:\n%s", name, cfg)
	return ""
}

func TestMultiRunnerConfig_PerEntryConcurrency(t *testing.T) {
	mr := &v1beta2.MultiRunner{
		ObjectMeta: metav1.ObjectMeta{Name: "m1", Namespace: "ns"},
		Spec: v1beta2.MultiRunnerSpec{
			GitlabInstanceURL: "https://gitlab.example.com",
			Concurrent:        30,
			Entries: []v1beta2.MultiRunnerEntry{
				{Name: "small", ConcurrencyLimits: v1beta2.ConcurrencyLimits{Limit: 2, RequestConcurrency: 1}},
				{Name: "big", ConcurrencyLimits: v1beta2.ConcurrencyLimits{Limit: 20, RequestConcurrency: 5}},
			},
		},
	}
	tokens := map[string]string{"small": "glrt-a", "big": "glrt-b"}

	cfg, _, err := MultiRunnerConfig(mr, tokens, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Assert inside each entry's own block: matching values against the whole
	// file passes even when the two entries have swapped them.
	for _, tc := range []struct {
		name  string
		wants []string
	}{
		{"small", []string{"\n  limit = 2\n", "\n  request_concurrency = 1\n"}},
		{"big", []string{"\n  limit = 20\n", "\n  request_concurrency = 5\n"}},
	} {
		block := runnerBlock(t, cfg, tc.name)
		for _, want := range tc.wants {
			if !strings.Contains(block, want) {
				t.Errorf("entry %q: expected %q, got block:\n%s", tc.name, want, block)
			}
		}
	}
}
