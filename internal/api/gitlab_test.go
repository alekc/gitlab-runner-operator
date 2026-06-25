package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"gitlab.k8s.alekc.dev/api/v1beta2"
)

// TestGitlabApi_CreateRunner is a smoke test: it exercises the client wiring
// against the real endpoint with a throwaway token and ignores the result.
func TestGitlabApi_CreateRunner(t *testing.T) {
	cl, _ := NewGitlabClient("9Bo36Uxwx6ay-cR-bCLh", "", nil)
	_, _ = cl.CreateRunner(v1beta2.RunnerCreateOptions{
		RunnerType: "instance_type",
		TagList:    []string{"gitlab-testing-operator"},
	})
}

func TestNewGitlabClient_RejectsInvalidCA(t *testing.T) {
	if _, err := NewGitlabClient("tok", "https://gitlab.example.com", []byte("not a pem")); err == nil {
		t.Fatal("expected an error for a CA bundle with no valid PEM certificate")
	}
}

func TestNewGitlabClient_AcceptsValidCA(t *testing.T) {
	cl, err := NewGitlabClient("tok", "https://gitlab.example.com", selfSignedCAPEM(t))
	if err != nil {
		t.Fatalf("unexpected error with a valid CA bundle: %v", err)
	}
	if cl == nil {
		t.Fatal("expected a client, got nil")
	}
}

// selfSignedCAPEM builds a throwaway self-signed CA certificate in PEM form.
func selfSignedCAPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
