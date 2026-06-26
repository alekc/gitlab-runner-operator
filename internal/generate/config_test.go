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
