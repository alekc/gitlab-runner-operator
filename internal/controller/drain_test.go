package controller

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// drainFixture is a Runner marked for deletion at deletedAt, with its manager
// Deployment, and a client holding both plus any extra pods.
func drainFixture(t *testing.T, deletedAt time.Time, pods ...*corev1.Pod) (*v1beta2.Runner, client.Client) {
	t.Helper()
	s := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{appsv1.AddToScheme, corev1.AddToScheme, v1beta2.AddToScheme} {
		if err := add(s); err != nil {
			t.Fatalf("build scheme: %v", err)
		}
	}
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{
		Name: "r1", Namespace: "ns", UID: "uid-r1",
		DeletionTimestamp: &metav1.Time{Time: deletedAt},
	}}
	r.Spec.RunnerTerminationGracePeriodSeconds = ptr.To[int64](600)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: r.ChildName(), Namespace: "ns"}}
	objs := []client.Object{dep}
	for _, p := range pods {
		objs = append(objs, p)
	}
	return r, fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()
}

func managerPod(name, ownerRS string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: "ns",
		Labels:          map[string]string{"deployment": "r1"},
		OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: ownerRS, APIVersion: "apps/v1", UID: "rs"}},
	}}
}

// A delete must not deregister from GitLab while the manager still runs jobs:
// the Deployment goes first, and drained stays false while its pod exists.
func TestDrainManagerWaitsForManagerPods(t *testing.T) {
	now := time.Now()
	r, cl := drainFixture(t, now, managerPod("p1", "gitlab-runner-r1-abc"))

	drained, err := drainManager(context.Background(), cl, cl, r, now.Add(time.Minute), logr.Discard())
	if err != nil {
		t.Fatalf("drainManager: %v", err)
	}
	if drained {
		t.Fatal("reported drained while a manager pod is still running")
	}
	err = cl.Get(context.Background(), client.ObjectKey{Namespace: "ns", Name: r.ChildName()}, &appsv1.Deployment{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("manager Deployment was not deleted: %v", err)
	}
}

// A pod that only shares the selector label belongs to someone else's workload.
func TestDrainManagerIgnoresForeignPods(t *testing.T) {
	now := time.Now()
	r, cl := drainFixture(t, now, managerPod("other", "unrelated-xyz"))

	drained, err := drainManager(context.Background(), cl, cl, r, now, logr.Discard())
	if err != nil {
		t.Fatalf("drainManager: %v", err)
	}
	if !drained {
		t.Fatal("waited on a pod that does not belong to the manager Deployment")
	}
}

// Past grace plus drainMargin the finalizer proceeds, so a pod stuck
// Terminating on a lost node cannot hold the object forever.
func TestDrainManagerGivesUpAfterTheGracePeriod(t *testing.T) {
	deletedAt := time.Now()
	r, cl := drainFixture(t, deletedAt, managerPod("p1", "gitlab-runner-r1-abc"))

	justBefore := deletedAt.Add(600*time.Second + drainMargin - time.Second)
	if drained, _ := drainManager(context.Background(), cl, cl, r, justBefore, logr.Discard()); drained {
		t.Fatal("gave up before the grace period plus margin ran out")
	}
	after := deletedAt.Add(600*time.Second + drainMargin + time.Second)
	if drained, _ := drainManager(context.Background(), cl, cl, r, after, logr.Discard()); !drained {
		t.Fatal("still waiting after the grace period plus margin")
	}
}

// The kubelet's clock starts when the pod is deleted, not when the CR was: an
// operator that was down for a while must still wait out the pod's own grace.
func TestDrainManagerWaitsOutAPodDeletedLate(t *testing.T) {
	crDeleted := time.Now()
	pod := managerPod("p1", "gitlab-runner-r1-abc")
	podDeadline := crDeleted.Add(30*time.Minute + 600*time.Second)
	pod.DeletionTimestamp = &metav1.Time{Time: podDeadline}
	pod.Finalizers = []string{"test/keep"}
	r, cl := drainFixture(t, crDeleted, pod)

	pastCRDeadline := crDeleted.Add(600*time.Second + drainMargin + time.Second)
	if drained, _ := drainManager(context.Background(), cl, cl, r, pastCRDeadline, logr.Discard()); drained {
		t.Fatal("gave up on the CR's deadline while the pod's own grace was still running")
	}
	pastPodDeadline := podDeadline.Add(drainMargin + time.Second)
	if drained, _ := drainManager(context.Background(), cl, cl, r, pastPodDeadline, logr.Discard()); !drained {
		t.Fatal("still waiting after the pod's own grace period plus margin")
	}
}

// Without a Deployment or pods, a repeat call must report drained, not error.
func TestDrainManagerIsIdempotent(t *testing.T) {
	now := time.Now()
	r, cl := drainFixture(t, now)
	for pass := 1; pass <= 2; pass++ {
		drained, err := drainManager(context.Background(), cl, cl, r, now, logr.Discard())
		if err != nil || !drained {
			t.Fatalf("pass %d: drained=%v err=%v", pass, drained, err)
		}
	}
}
