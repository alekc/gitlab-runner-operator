package validate

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeployment_CustomCAMount(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1beta2.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	newRunner := func(ca *v1beta2.CASource) *v1beta2.Runner {
		return &v1beta2.Runner{
			TypeMeta:   metav1.TypeMeta{APIVersion: "gitlab.k8s.alekc.dev/v1beta2", Kind: "Runner"},
			ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns", UID: "uid-1"},
			Spec:       v1beta2.RunnerSpec{CACertificate: ca},
		}
	}

	getDeployment := func(t *testing.T, runner *v1beta2.Runner) appsv1.Deployment {
		t.Helper()
		cl := fake.NewClientBuilder().WithScheme(scheme).Build()
		if _, err := Deployment(context.Background(), cl, runner, logr.Discard()); err != nil {
			t.Fatalf("Deployment: %v", err)
		}
		var dep appsv1.Deployment
		key := client.ObjectKey{Namespace: "ns", Name: runner.ChildName()}
		if err := cl.Get(context.Background(), key, &dep); err != nil {
			t.Fatalf("deployment not created: %v", err)
		}
		return dep
	}

	t.Run("no CA leaves no custom-ca volume", func(t *testing.T) {
		dep := getDeployment(t, newRunner(nil))
		if hasVolume(dep, "custom-ca") {
			t.Fatal("did not expect a custom-ca volume without a CA")
		}
	})

	t.Run("configmap CA adds volume and read-only mount", func(t *testing.T) {
		dep := getDeployment(t, newRunner(&v1beta2.CASource{
			ConfigMapKeyRef: &v1beta2.CAKeyRef{Name: "ca-cm"},
		}))
		if !hasVolume(dep, "custom-ca") {
			t.Fatal("expected a custom-ca volume")
		}
		if !hasMount(dep.Spec.Template.Spec.Containers[0], "custom-ca", types.CACertMountDir) {
			t.Fatalf("expected custom-ca mounted read-only at %s", types.CACertMountDir)
		}
	})

	t.Run("secret CA adds volume and read-only mount", func(t *testing.T) {
		dep := getDeployment(t, newRunner(&v1beta2.CASource{
			SecretKeyRef: &v1beta2.CAKeyRef{Name: "ca-secret"},
		}))
		if !hasVolume(dep, "custom-ca") {
			t.Fatal("expected a custom-ca volume")
		}
		if !hasMount(dep.Spec.Template.Spec.Containers[0], "custom-ca", types.CACertMountDir) {
			t.Fatalf("expected custom-ca mounted read-only at %s", types.CACertMountDir)
		}
	})
}

func hasVolume(dep appsv1.Deployment, name string) bool {
	for _, v := range dep.Spec.Template.Spec.Volumes {
		if v.Name == name {
			return true
		}
	}
	return false
}

func hasMount(c corev1.Container, name, path string) bool {
	for _, m := range c.VolumeMounts {
		if m.Name == name && m.MountPath == path && m.ReadOnly {
			return true
		}
	}
	return false
}
