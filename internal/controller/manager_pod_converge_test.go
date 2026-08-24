package controller

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/validate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// The reconcile must settle on the second pass, against a real apiserver and
// through the real renderer. A fake client defaults nothing and prunes nothing,
// so it cannot fail for the reason that matters: the apiserver storing
// something other than what the operator sent.
var _ = Describe("manager pod convergence", func() {
	reconcile := func(runner *v1beta2.Runner) bool {
		GinkgoHelper()
		res, err := validate.Deployment(ctx, k8sClient, runner, logr.Discard())
		Expect(err).NotTo(HaveOccurred())
		return res != nil
	}

	storedVersion := func(name string) string {
		GinkgoHelper()
		var deployment appsv1.Deployment
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: "gitlab-runner-" + name, Namespace: "default"}, &deployment)).To(Succeed())
		return deployment.ResourceVersion
	}

	// Settling has to hold on every later pass, not just the second. Where the
	// cluster refuses part of the spec the reconcile keeps issuing an update it
	// knows will not stick, so what makes that harmless is the write being a
	// no-op: an unchanged ResourceVersion means no watch event, so the operator
	// does not wake itself in a loop.
	settles := func(name string, spec v1beta2.RunnerSpec) {
		GinkgoHelper()
		runner := &v1beta2.Runner{
			TypeMeta:   metav1.TypeMeta{APIVersion: "gitlab.k8s.alekc.dev/v1beta2", Kind: "Runner"},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID(name)},
			Spec:       spec,
		}
		Expect(reconcile(runner)).To(BeTrue(), "the first pass must create the deployment")
		Expect(reconcile(runner)).To(BeFalse(), "the reconcile did not settle: it would requeue forever and never report ready")

		settled := storedVersion(name)
		for pass := 3; pass <= 5; pass++ {
			Expect(reconcile(runner)).To(BeFalse(), "pass %d stopped being settled", pass)
			Expect(storedVersion(name)).To(Equal(settled), "pass %d moved the ResourceVersion, so it would wake itself again", pass)
		}
	}

	It("settles with no placement", func() {
		settles("cv-bare", v1beta2.RunnerSpec{})
	})

	It("settles with rich placement", func() {
		settles("cv-rich", v1beta2.RunnerSpec{
			RunnerNodeSelector: map[string]string{"node-pool": "ci"},
			RunnerTolerations: []corev1.Toleration{{
				Key:               "node.kubernetes.io/not-ready",
				Operator:          corev1.TolerationOpExists,
				Effect:            corev1.TaintEffectNoExecute,
				TolerationSeconds: ptr.To[int64](300),
			}},
			RunnerAffinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
						Weight: 10,
						Preference: corev1.NodeSelectorTerm{
							MatchExpressions: []corev1.NodeSelectorRequirement{{
								Key:      "node-lifecycle",
								Operator: corev1.NodeSelectorOpNotIn,
								Values:   []string{"spot"},
							}},
						},
					}},
				},
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
						TopologyKey:   "kubernetes.io/hostname",
						LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"deployment": "cv-rich"}},
					}},
				},
			},
			// A priority class that does not exist: the Deployment is still
			// stored, so this must converge rather than retry forever.
			RunnerPriorityClassName: "no-such-priority-class",
		})
	})

	It("settles with explicit resources, security context and env", func() {
		settles("cv-container", v1beta2.RunnerSpec{
			RunnerResources: &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
			},
			RunnerSecurityContext: &corev1.SecurityContext{
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				Capabilities: &corev1.Capabilities{
					Add:  []corev1.Capability{"NET_BIND_SERVICE"},
					Drop: []corev1.Capability{"ALL"},
				},
			},
			// A plain value plus a secretKeyRef, so both branches of EnvVar's
			// schema round-trip, not just the simpler one.
			RunnerEnv: []corev1.EnvVar{
				{Name: "HTTP_PROXY", Value: "http://proxy:3128"},
				{Name: "PROXY_PASSWORD", ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "proxy-creds"},
						Key:                  "password",
					},
				}},
			},
		})
	})

	It("settles with empty structs", func() {
		settles("cv-empty", v1beta2.RunnerSpec{
			RunnerNodeSelector:    map[string]string{},
			RunnerTolerations:     []corev1.Toleration{},
			RunnerAffinity:        &corev1.Affinity{},
			RunnerResources:       &corev1.ResourceRequirements{},
			RunnerSecurityContext: &corev1.SecurityContext{},
			RunnerEnv:             []corev1.EnvVar{},
		})
	})

	// The case that broke the first implementation. resources.claims is behind
	// the DynamicResourceAllocation gate, so this CRD accepts it and the
	// apiserver drops it from the pod template. The compare can never match, so
	// the reconcile has to notice the update stored nothing and settle anyway.
	It("settles when the cluster drops a field the CRD accepts", func() {
		settles("cv-dropped", v1beta2.RunnerSpec{
			RunnerResources: &corev1.ResourceRequirements{
				Claims: []corev1.ResourceClaim{{Name: "gpu"}},
			},
		})
	})

	// A runner_image change has to roll the manager, which is the most
	// consequential thing this comparison does. It used to have its own branch.
	It("rolls when the runner image changes", func() {
		runner := &v1beta2.Runner{
			TypeMeta:   metav1.TypeMeta{APIVersion: "gitlab.k8s.alekc.dev/v1beta2", Kind: "Runner"},
			ObjectMeta: metav1.ObjectMeta{Name: "cv-image", Namespace: "default", UID: "cv-image"},
		}
		res, err := validate.Deployment(ctx, k8sClient, runner, logr.Discard())
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		res, err = validate.Deployment(ctx, k8sClient, runner, logr.Discard())
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(BeNil(), "precondition: must be settled before the image change")

		runner.Spec.RunnerImage = "gitlab/gitlab-runner:alpine-v19.4.0"
		res, err = validate.Deployment(ctx, k8sClient, runner, logr.Discard())
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil(), "an image change must roll the manager pod")
	})
})
