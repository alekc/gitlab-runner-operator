package validate

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestSecret_CAKey verifies the resolved CA bundle is written into the config
// Secret under types.CACertFileName (so it is mounted into the runner via the
// existing config volume), and is absent when no CA is configured.
func TestSecret_CAKey(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1beta2.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	runner := &v1beta2.Runner{
		TypeMeta:   metav1.TypeMeta{APIVersion: "gitlab.k8s.alekc.dev/v1beta2", Kind: "Runner"},
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns", UID: "uid-1"},
	}

	getSecret := func(t *testing.T, caPEM []byte) corev1.Secret {
		t.Helper()
		cl := fake.NewClientBuilder().WithScheme(scheme).Build()
		_, err := Secret(context.Background(), cl, runner, logr.Discard(), "[[runners]]\n", map[string]string{"r1": "glrt-x"}, caPEM)
		if err != nil {
			t.Fatalf("Secret: %v", err)
		}
		var s corev1.Secret
		key := client.ObjectKey{Namespace: "ns", Name: runner.ChildName()}
		if err := cl.Get(context.Background(), key, &s); err != nil {
			t.Fatalf("config secret not created: %v", err)
		}
		return s
	}

	t.Run("no CA leaves no ca.crt key", func(t *testing.T) {
		s := getSecret(t, nil)
		if _, ok := s.Data[types.CACertFileName]; ok {
			t.Fatal("did not expect a ca.crt key without a CA")
		}
	})

	t.Run("CA present is stored under ca.crt", func(t *testing.T) {
		pem := []byte("-----BEGIN CERTIFICATE-----\nMIIBfake\n-----END CERTIFICATE-----\n")
		s := getSecret(t, pem)
		got, ok := s.Data[types.CACertFileName]
		if !ok {
			t.Fatal("expected a ca.crt key in the config secret")
		}
		if string(got) != string(pem) {
			t.Fatalf("ca.crt = %q, want %q", got, pem)
		}
	})
}
