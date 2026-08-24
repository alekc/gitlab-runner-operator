package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.k8s.alekc.dev/internal/validate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// validate.Deployment rolls the manager pod when the live pod shape differs
// from the rendered one. That only converges while the apiserver returns these
// fields as they were sent: a release that starts defaulting one would make
// every reconcile see a difference and re-apply forever, never reaching Ready.
// This is the assumption, asserted against a real apiserver.
var _ = Describe("manager pod shape", func() {
	shapedTemplate := func(placement corev1.PodSpec) corev1.PodTemplateSpec {
		spec := placement
		spec.Containers = []corev1.Container{{
			Name:            "runner",
			Image:           "gitlab/gitlab-runner:alpine-v19.3.0",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
			},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(true),
				ReadOnlyRootFilesystem:   ptr.To(true),
				Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			},
		}}
		return corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"deployment": "shape"}},
			Spec:       spec,
		}
	}

	survivesRoundTrip := func(name string, placement corev1.PodSpec) {
		template := shapedTemplate(placement)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.To[int32](1),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"deployment": "shape"}},
				Template: template,
			},
		}
		Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

		var stored appsv1.Deployment
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: "default"}, &stored)).To(Succeed())

		sent := validate.ManagerPodShape(&template)
		got := validate.ManagerPodShape(&stored.Spec.Template)
		Expect(apiequality.Semantic.DeepEqual(sent, got)).To(BeTrue(),
			"apiserver changed the manager pod shape, so the reconcile would never converge:\nsent: %+v\ngot:  %+v", sent, got)
	}

	It("survives an apiserver round-trip with placement set", func() {
		survivesRoundTrip("shape-placed", corev1.PodSpec{
			NodeSelector: map[string]string{"node-pool": "ci", "kubernetes.io/arch": "arm64"},
			Tolerations: []corev1.Toleration{{
				Key:      "dedicated",
				Operator: corev1.TolerationOpEqual,
				Value:    "ci",
				Effect:   corev1.TaintEffectNoSchedule,
			}},
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{{
							MatchExpressions: []corev1.NodeSelectorRequirement{{
								Key:      "node-lifecycle",
								Operator: corev1.NodeSelectorOpNotIn,
								Values:   []string{"spot"},
							}},
						}},
					},
				},
			},
			PriorityClassName: "system-cluster-critical",
		})
	})

	It("survives an apiserver round-trip with no placement", func() {
		survivesRoundTrip("shape-bare", corev1.PodSpec{})
	})

	// An empty map or list is dropped by omitempty and read back as nil. The
	// comparison relies on apiequality.Semantic equating the two.
	It("survives an apiserver round-trip with empty collections", func() {
		survivesRoundTrip("shape-empty", corev1.PodSpec{
			NodeSelector: map[string]string{},
			Tolerations:  []corev1.Toleration{},
		})
	})
})
