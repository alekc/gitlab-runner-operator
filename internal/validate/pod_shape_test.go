package validate

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func tplWith(spec corev1.PodSpec) *corev1.PodTemplateSpec {
	return &corev1.PodTemplateSpec{Spec: spec}
}

// A sidecar injected ahead of the runner container must not be mistaken for it.
// Reading Containers[0] here would compare the sidecar's image against the
// runner image on every reconcile, which never converges.
func TestManagerPodShape_FindsRunnerContainerByName(t *testing.T) {
	tpl := tplWith(corev1.PodSpec{Containers: []corev1.Container{
		{Name: "istio-proxy", Image: "proxy:1", ImagePullPolicy: corev1.PullAlways},
		{Name: "runner", Image: "gitlab/gitlab-runner:v19", ImagePullPolicy: corev1.PullIfNotPresent},
	}})

	shape := ManagerPodShape(tpl)
	if shape.Image != "gitlab/gitlab-runner:v19" {
		t.Fatalf("image: got %q, want the runner container's", shape.Image)
	}
	if shape.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Fatalf("pull policy: got %q, want the runner container's", shape.ImagePullPolicy)
	}
}

// No runner container leaves the container fields zero, which cannot match a
// rendered spec, so the reconcile treats it as a change and repairs it.
func TestManagerPodShape_MissingRunnerContainerIsZero(t *testing.T) {
	tpl := tplWith(corev1.PodSpec{Containers: []corev1.Container{
		{Name: "istio-proxy", Image: "proxy:1"},
	}})

	if shape := ManagerPodShape(tpl); shape.Image != "" {
		t.Fatalf("image: got %q, want empty", shape.Image)
	}
}

// NodeSelector and Tolerations are omitempty: a spec asking for an empty map or
// list sends nothing and reads back nil. apiequality.Semantic equates the two,
// which is why the comparison needs no normalising. Switching it to
// reflect.DeepEqual would roll the manager pod forever, and fail here first.
func TestManagerPodShape_EmptyComparesEqualToUnset(t *testing.T) {
	empty := ManagerPodShape(tplWith(corev1.PodSpec{
		NodeSelector: map[string]string{},
		Tolerations:  []corev1.Toleration{},
	}))
	unset := ManagerPodShape(tplWith(corev1.PodSpec{}))

	if !apiequality.Semantic.DeepEqual(empty, unset) {
		t.Fatalf("empty and unset must compare equal:\n empty: %+v\n unset: %+v", empty, unset)
	}
}

// Semantic equality is what makes equivalent quantities a non-event. Written
// as 1000m and as 1 they are the same request, and rolling the manager pod
// because a user reformatted a number would lose its in-flight jobs.
func TestManagerPodShape_EquivalentQuantitiesCompareEqual(t *testing.T) {
	shape := func(cpu string) PodShape {
		return ManagerPodShape(tplWith(corev1.PodSpec{Containers: []corev1.Container{{
			Name: "runner",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: mustQuantity(t, cpu)},
			},
		}}}))
	}

	if !apiequality.Semantic.DeepEqual(shape("1000m"), shape("1")) {
		t.Fatal("1000m and 1 must compare equal")
	}
	if apiequality.Semantic.DeepEqual(shape("1"), shape("2")) {
		t.Fatal("1 and 2 must not compare equal")
	}
}

// A genuine placement change must be visible to the comparison, otherwise the
// field is accepted and inert.
func TestManagerPodShape_PlacementChangeIsVisible(t *testing.T) {
	base := corev1.PodSpec{NodeSelector: map[string]string{"node-pool": "ci"}}
	moved := corev1.PodSpec{NodeSelector: map[string]string{"node-pool": "build"}}
	tainted := corev1.PodSpec{
		NodeSelector: map[string]string{"node-pool": "ci"},
		Tolerations:  []corev1.Toleration{{Key: "dedicated", Value: "ci"}},
	}
	pinned := corev1.PodSpec{
		NodeSelector:      map[string]string{"node-pool": "ci"},
		PriorityClassName: "system-cluster-critical",
	}
	affine := corev1.PodSpec{
		NodeSelector: map[string]string{"node-pool": "ci"},
		Affinity:     &corev1.Affinity{NodeAffinity: &corev1.NodeAffinity{}},
	}

	for name, changed := range map[string]corev1.PodSpec{
		"node selector":  moved,
		"tolerations":    tainted,
		"priority class": pinned,
		"affinity":       affine,
	} {
		t.Run(name, func(t *testing.T) {
			if apiequality.Semantic.DeepEqual(ManagerPodShape(tplWith(base)), ManagerPodShape(tplWith(changed))) {
				t.Fatalf("a %s change must not compare equal", name)
			}
		})
	}
}

// SecurityContext is a pointer, so an unset one and an explicitly empty one are
// different shapes. Both are compared, neither is normalised.
func TestManagerPodShape_SecurityContextPointerIsCompared(t *testing.T) {
	with := ManagerPodShape(tplWith(corev1.PodSpec{Containers: []corev1.Container{{
		Name:            "runner",
		SecurityContext: &corev1.SecurityContext{RunAsNonRoot: ptr.To(true)},
	}}}))
	without := ManagerPodShape(tplWith(corev1.PodSpec{Containers: []corev1.Container{{
		Name: "runner",
	}}}))

	if apiequality.Semantic.DeepEqual(with, without) {
		t.Fatal("a security context change must not compare equal")
	}
}

func mustQuantity(t *testing.T, s string) resource.Quantity {
	t.Helper()
	q, err := resource.ParseQuantity(s)
	if err != nil {
		t.Fatalf("ParseQuantity(%q): %v", s, err)
	}
	return q
}
