package controller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/crud"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// enforceAllowedBuildNamespaces must prune only the disallowed namespace. The
// envtest case for this moves a Runner's single namespace, so a filtered and an
// unfiltered keep set both prune the one binding and the assertion cannot tell
// them apart. Two namespaces, one still allowed, is what distinguishes them.
func TestEnforceAllowedBuildNamespacesPrunesOnlyTheDisallowedOne(t *testing.T) {
	const (
		ownNS     = "rns"
		allowedNS = "build-allowed"
		deniedNS  = "build-denied"
	)
	s := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{rbacv1.AddToScheme, corev1.AddToScheme, v1beta2.AddToScheme} {
		if err := add(s); err != nil {
			t.Fatalf("build scheme: %v", err)
		}
	}

	mr := &v1beta2.MultiRunner{ObjectMeta: metav1.ObjectMeta{Name: "m", Namespace: ownNS, UID: "uid-m"}}
	a := v1beta2.MultiRunnerEntry{Name: "a"}
	a.ExecutorConfig.Namespace = allowedNS
	b := v1beta2.MultiRunnerEntry{Name: "b"}
	b.ExecutorConfig.Namespace = deniedNS
	mr.Spec.Entries = []v1beta2.MultiRunnerEntry{a, b}

	cl := fake.NewClientBuilder().WithScheme(s).Build()
	ctx := context.Background()

	// Provision RBAC for both namespaces first, so the bindings carry the real
	// labels rather than a copy of them this test would have to keep in sync.
	// This is the pre-tightening state the prune is meant to clean up.
	if err := crud.CreateRBACIfMissing(ctx, cl, cl, mr, logr.Discard()); err != nil {
		t.Fatalf("seed RBAC: %v", err)
	}
	for _, ns := range []string{allowedNS, deniedNS} {
		if !bindingPresent(t, cl, ns, mr.ChildName()) {
			t.Fatalf("seed failed: no binding in %s", ns)
		}
	}

	ok, err := enforceAllowedBuildNamespaces(ctx, cl, cl, mr, []string{allowedNS}, logr.Discard())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("a disallowed namespace was accepted")
	}
	if bindingPresent(t, cl, deniedNS, mr.ChildName()) {
		t.Error("the binding in the disallowed namespace survived; a tightened policy would not take effect")
	}
	if !bindingPresent(t, cl, allowedNS, mr.ChildName()) {
		t.Error("the binding in the still-allowed namespace was pruned")
	}
}

func bindingPresent(t *testing.T, cl client.Client, namespace, name string) bool {
	t.Helper()
	var rb rbacv1.RoleBinding
	err := cl.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, &rb)
	if err != nil && !errors.IsNotFound(err) {
		t.Fatalf("get rolebinding %s/%s: %v", namespace, name, err)
	}
	return err == nil
}
