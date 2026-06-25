package controller

import (
	"context"
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestResolveCABundle covers reading the CA PEM from a Secret or a ConfigMap,
// the default and custom key, the empty source, and the error paths.
func TestResolveCABundle(t *testing.T) {
	const ns = "default"
	const pemBody = "-----BEGIN CERTIFICATE-----\nMIIBfake\n-----END CERTIFICATE-----\n"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-secret", Namespace: ns},
		Data: map[string][]byte{
			"ca.crt": []byte(pemBody),
			"custom": []byte("custom-secret-ca"),
		},
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-cm", Namespace: ns},
		Data: map[string]string{
			"ca.crt": pemBody,
			"custom": "custom-cm-ca",
		},
	}
	build := func() client.Client {
		return fake.NewClientBuilder().WithObjects(secret, cm).Build()
	}

	cases := []struct {
		name    string
		src     *v1beta2.CASource
		want    string
		wantErr bool
	}{
		{name: "nil source", src: nil, want: ""},
		{name: "empty source", src: &v1beta2.CASource{}, want: ""},
		{
			name: "secret default key",
			src:  &v1beta2.CASource{SecretKeyRef: &v1beta2.CAKeyRef{Name: "ca-secret"}},
			want: pemBody,
		},
		{
			name: "secret custom key",
			src:  &v1beta2.CASource{SecretKeyRef: &v1beta2.CAKeyRef{Name: "ca-secret", Key: "custom"}},
			want: "custom-secret-ca",
		},
		{
			name: "configmap default key",
			src:  &v1beta2.CASource{ConfigMapKeyRef: &v1beta2.CAKeyRef{Name: "ca-cm"}},
			want: pemBody,
		},
		{
			name: "configmap custom key",
			src:  &v1beta2.CASource{ConfigMapKeyRef: &v1beta2.CAKeyRef{Name: "ca-cm", Key: "custom"}},
			want: "custom-cm-ca",
		},
		{
			name:    "secret missing key",
			src:     &v1beta2.CASource{SecretKeyRef: &v1beta2.CAKeyRef{Name: "ca-secret", Key: "absent"}},
			wantErr: true,
		},
		{
			name:    "configmap missing key",
			src:     &v1beta2.CASource{ConfigMapKeyRef: &v1beta2.CAKeyRef{Name: "ca-cm", Key: "absent"}},
			wantErr: true,
		},
		{
			name:    "missing secret",
			src:     &v1beta2.CASource{SecretKeyRef: &v1beta2.CAKeyRef{Name: "absent"}},
			wantErr: true,
		},
		{
			name:    "missing configmap",
			src:     &v1beta2.CASource{ConfigMapKeyRef: &v1beta2.CAKeyRef{Name: "absent"}},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveCABundle(context.Background(), build(), ns, tc.src)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value %q)", string(got))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("got %q, want %q", string(got), tc.want)
			}
		})
	}
}

// TestCASourceValidate covers the webhook-level validation rules.
func TestCASourceValidate(t *testing.T) {
	cases := []struct {
		name    string
		src     *v1beta2.CASource
		wantErr bool
	}{
		{name: "nil", src: nil},
		{name: "empty", src: &v1beta2.CASource{}},
		{name: "secret ok", src: &v1beta2.CASource{SecretKeyRef: &v1beta2.CAKeyRef{Name: "s"}}},
		{name: "configmap ok", src: &v1beta2.CASource{ConfigMapKeyRef: &v1beta2.CAKeyRef{Name: "c"}}},
		{name: "secret no name", src: &v1beta2.CASource{SecretKeyRef: &v1beta2.CAKeyRef{}}, wantErr: true},
		{name: "configmap no name", src: &v1beta2.CASource{ConfigMapKeyRef: &v1beta2.CAKeyRef{}}, wantErr: true},
		{
			name:    "both set",
			src:     &v1beta2.CASource{SecretKeyRef: &v1beta2.CAKeyRef{Name: "s"}, ConfigMapKeyRef: &v1beta2.CAKeyRef{Name: "c"}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.src.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
