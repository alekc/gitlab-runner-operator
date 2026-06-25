package controller

import (
	"context"
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestResolveTokenSource covers the two supply modes (inline value and Secret
// key reference) plus the default-key, custom-key, and optional behaviours.
func TestResolveTokenSource(t *testing.T) {
	const ns = "default"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: ns},
		Data: map[string][]byte{
			"token":  []byte("default-key-token"),
			"custom": []byte("custom-key-token"),
		},
	}
	build := func() client.Client {
		return fake.NewClientBuilder().WithObjects(secret).Build()
	}

	cases := []struct {
		name    string
		src     *v1beta2.TokenSource
		want    string
		wantErr bool
	}{
		{name: "nil source", src: nil, want: ""},
		{name: "inline value", src: &v1beta2.TokenSource{Value: "inline"}, want: "inline"},
		{
			name: "secret default key",
			src:  &v1beta2.TokenSource{SecretKeyRef: &v1beta2.SecretKeySelector{Name: "creds"}},
			want: "default-key-token",
		},
		{
			name: "secret custom key",
			src:  &v1beta2.TokenSource{SecretKeyRef: &v1beta2.SecretKeySelector{Name: "creds", Key: "custom"}},
			want: "custom-key-token",
		},
		{
			name:    "missing key not optional",
			src:     &v1beta2.TokenSource{SecretKeyRef: &v1beta2.SecretKeySelector{Name: "creds", Key: "absent"}},
			wantErr: true,
		},
		{
			name: "missing key optional",
			src:  &v1beta2.TokenSource{SecretKeyRef: &v1beta2.SecretKeySelector{Name: "creds", Key: "absent", Optional: ptr.To(true)}},
			want: "",
		},
		{
			name:    "missing secret not optional",
			src:     &v1beta2.TokenSource{SecretKeyRef: &v1beta2.SecretKeySelector{Name: "absent"}},
			wantErr: true,
		},
		{
			name: "missing secret optional",
			src:  &v1beta2.TokenSource{SecretKeyRef: &v1beta2.SecretKeySelector{Name: "absent", Optional: ptr.To(true)}},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveTokenSource(context.Background(), build(), ns, tc.src)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
