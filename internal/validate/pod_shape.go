package validate

import (
	"encoding/json"
	"fmt"

	"gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
)

// PodShape is the slice of the runner manager pod the operator sets from the
// CR. Deployment compares this rather than the whole pod spec, because the
// apiserver defaults fields the operator never sets and a whole-spec diff
// therefore never converges.
type PodShape struct {
	Image             string
	ImagePullPolicy   corev1.PullPolicy
	Resources         corev1.ResourceRequirements
	SecurityContext   *corev1.SecurityContext
	Env               []corev1.EnvVar
	NodeSelector      map[string]string
	Tolerations       []corev1.Toleration
	Affinity          *corev1.Affinity
	PriorityClassName string
}

// ManagerPodShape reads the shape out of a pod template. Compare two of these
// with apiequality.Semantic, which equates nil and empty. The "manager pod
// convergence" specs in internal/controller pin the round-trip against a real
// apiserver for the values the operator renders, not for every possible input:
// Deployment settles a difference the cluster refuses to store.
func ManagerPodShape(tpl *corev1.PodTemplateSpec) PodShape {
	shape := PodShape{
		NodeSelector:      tpl.Spec.NodeSelector,
		Tolerations:       tpl.Spec.Tolerations,
		Affinity:          tpl.Spec.Affinity,
		PriorityClassName: tpl.Spec.PriorityClassName,
	}
	// A template with no runner container leaves the container fields zero,
	// which never matches a rendered spec and so forces an update.
	for i := range tpl.Spec.Containers {
		c := &tpl.Spec.Containers[i]
		if c.Name != types.RunnerContainerName {
			continue
		}
		shape.Image = c.Image
		shape.ImagePullPolicy = c.ImagePullPolicy
		shape.Resources = c.Resources
		shape.SecurityContext = c.SecurityContext
		shape.Env = c.Env
		break
	}
	return shape
}

// shapeJSON renders a shape for a log line. The struct holds pointers, which
// would otherwise log as addresses and tell the reader nothing.
func shapeJSON(shape PodShape) string {
	encoded, err := json.Marshal(shape)
	if err != nil {
		return fmt.Sprintf("%+v", shape)
	}
	return string(encoded)
}
