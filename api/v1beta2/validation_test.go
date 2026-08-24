package v1beta2

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func valMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: "default"}
}

// unstructuredRunner builds a minimal valid Runner with one spec field forced,
// so a value omitempty would drop can still be submitted.
func unstructuredRunner(name, field string, value any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": GroupVersion.String(),
		"kind":       "Runner",
		"metadata":   map[string]any{"name": name, "namespace": "default"},
		"spec": map[string]any{
			"authentication": map[string]any{
				"token": map[string]any{"value": "glrt-x"},
			},
			field: value,
		},
	}}
}

// expectStoredZero asserts the apiserver kept the field at zero rather than
// pruning it, which a misspelled field name would do without erroring.
func expectStoredZero(u *unstructured.Unstructured, field string) {
	GinkgoHelper()
	v, found, err := unstructured.NestedInt64(u.Object, "spec", field)
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue(), "spec.%s was pruned, not stored", field)
	Expect(v).To(BeZero())
}

func valByoAuth() GitlabAuth {
	return GitlabAuth{Token: &TokenSource{Value: "glrt-x"}}
}

// expectInvalid asserts the apiserver rejected the create as Invalid (a schema
// or CEL rule) for exactly one reason, and that it is the expected one.
// Matching err.Error() instead searches all causes concatenated, so an input
// that trips two rules would still satisfy a spec naming only one of them.
func expectInvalid(err error, wantMsg string) {
	GinkgoHelper()
	Expect(err).To(HaveOccurred(), "expected the apiserver to reject: %s", wantMsg)
	Expect(apierrors.IsInvalid(err)).To(BeTrue(), "want an Invalid (schema/CEL) rejection, got: %v", err)

	var status apierrors.APIStatus
	Expect(errors.As(err, &status)).To(BeTrue(), "want an APIStatus error, got: %v", err)
	Expect(status.Status().Details).NotTo(BeNil(), "want status details, got: %v", err)

	// When a structural error (enum, required) coexists with a CEL rule, the
	// apiserver appends a boilerplate cause saying the rules were not evaluated.
	// It is not a second rule, so drop it before pinning the count.
	var causes []metav1.StatusCause
	for _, c := range status.Status().Details.Causes {
		if strings.Contains(c.Message, "some validation rules were not checked") {
			continue
		}
		causes = append(causes, c)
	}
	Expect(causes).To(HaveLen(1), "want exactly one cause so the spec pins one rule, got: %v", causes)
	Expect(causes[0].Message).To(ContainSubstring(wantMsg))
}

var _ = Describe("CRD validation", func() {
	It("applies the gitlab_instance_url default on create", func() {
		r := &Runner{ObjectMeta: valMeta("val-default"), Spec: RunnerSpec{Authentication: valByoAuth()}}
		Expect(k8sClient.Create(ctx, r)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, r) })
		Expect(r.Spec.GitlabInstanceURL).To(Equal("https://gitlab.com/"))
	})

	It("defaults neither concurrency field on create", func() {
		r := &Runner{ObjectMeta: valMeta("val-conc-default"), Spec: RunnerSpec{Authentication: valByoAuth()}}
		Expect(k8sClient.Create(ctx, r)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, r) })
		// A default on either would render a key the spec never asked for, which
		// is the ceiling-invention this API deliberately gave up.
		Expect(r.Spec.Limit).To(BeZero())
		Expect(r.Spec.RequestConcurrency).To(BeZero())
	})

	It("defaults neither concurrency field on a MultiRunner entry", func() {
		m := &MultiRunner{ObjectMeta: valMeta("val-conc-entry"), Spec: MultiRunnerSpec{
			Entries: []MultiRunnerEntry{{Name: "e1", Authentication: valByoAuth()}},
		}}
		Expect(k8sClient.Create(ctx, m)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, m) })
		Expect(m.Spec.Entries[0].Limit).To(BeZero())
		Expect(m.Spec.Entries[0].RequestConcurrency).To(BeZero())
	})

	// omitempty drops a typed zero before it reaches the apiserver, so the
	// boundary itself can only be submitted unstructured.
	It("accepts an explicit zero limit, meaning bounded by concurrent", func() {
		r := unstructuredRunner("val-conc-zero-limit", "limit", int64(0))
		Expect(k8sClient.Create(ctx, r)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, r) })
		expectStoredZero(r, "limit")
	})

	It("rejects a negative limit", func() {
		r := &Runner{ObjectMeta: valMeta("val-limit-negative"), Spec: RunnerSpec{
			Authentication:    valByoAuth(),
			ConcurrencyLimits: ConcurrencyLimits{Limit: -1},
		}}
		expectInvalid(k8sClient.Create(ctx, r), "should be greater than or equal to 0")
	})

	It("accepts an explicit zero request_concurrency, deferring to upstream", func() {
		r := unstructuredRunner("val-conc-zero-rc", "request_concurrency", int64(0))
		Expect(k8sClient.Create(ctx, r)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, r) })
		expectStoredZero(r, "request_concurrency")
	})

	It("rejects a negative request_concurrency", func() {
		r := &Runner{ObjectMeta: valMeta("val-rc-negative"), Spec: RunnerSpec{
			Authentication:    valByoAuth(),
			ConcurrencyLimits: ConcurrencyLimits{RequestConcurrency: -1},
		}}
		expectInvalid(k8sClient.Create(ctx, r), "should be greater than or equal to 0")
	})

	It("accepts a valid managed Runner", func() {
		r := &Runner{ObjectMeta: valMeta("val-managed"), Spec: RunnerSpec{Authentication: GitlabAuth{
			AccessToken:   &TokenSource{Value: "glpat-x"},
			CreateOptions: &RunnerCreateOptions{RunnerType: "instance_type"},
		}}}
		Expect(k8sClient.Create(ctx, r)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, r) })
	})

	// Positive controls for the conditional rules. Without these, inverting a
	// rule so it rejects every input still passes the whole suite.
	It("accepts a single caCertificate source", func() {
		r := &Runner{ObjectMeta: valMeta("val-ca-one"), Spec: RunnerSpec{
			Authentication: valByoAuth(),
			CACertificate:  &CASource{ConfigMapKeyRef: &CAKeyRef{Name: "ca-cm"}},
		}}
		Expect(k8sClient.Create(ctx, r)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, r) })
	})

	It("accepts a project_type runner with project_id", func() {
		r := &Runner{ObjectMeta: valMeta("val-project-ok"), Spec: RunnerSpec{Authentication: GitlabAuth{
			AccessToken: &TokenSource{Value: "glpat-x"},
			CreateOptions: &RunnerCreateOptions{
				RunnerType: "project_type",
				ProjectID:  new(42),
			},
		}}}
		Expect(k8sClient.Create(ctx, r)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, r) })
	})

	DescribeTable("rejects an invalid Runner with a CEL message",
		func(name string, spec RunnerSpec, wantMsg string) {
			expectInvalid(k8sClient.Create(ctx, &Runner{ObjectMeta: valMeta(name), Spec: spec}), wantMsg)
		},
		Entry("both auth modes set", "val-both", RunnerSpec{Authentication: GitlabAuth{
			Token:         &TokenSource{Value: "glrt-x"},
			AccessToken:   &TokenSource{Value: "glpat-x"},
			CreateOptions: &RunnerCreateOptions{RunnerType: "instance_type"},
		}}, "not both"),
		Entry("no auth mode set", "val-none", RunnerSpec{}, "one of token or create_options must be set"),
		Entry("create_options without access_token", "val-noat", RunnerSpec{Authentication: GitlabAuth{
			CreateOptions: &RunnerCreateOptions{RunnerType: "instance_type"},
		}}, "create_options requires access_token"),
		// token is set so the "one of token or create_options" rule passes, leaving
		// this input to trip the access_token rule alone.
		Entry("access_token without create_options", "val-atnoco", RunnerSpec{Authentication: GitlabAuth{
			Token:       &TokenSource{Value: "glrt-x"},
			AccessToken: &TokenSource{Value: "glpat-x"},
		}}, "access_token is only used with create_options"),
		Entry("group_type without group_id", "val-nogid", RunnerSpec{Authentication: GitlabAuth{
			AccessToken:   &TokenSource{Value: "x"},
			CreateOptions: &RunnerCreateOptions{RunnerType: "group_type"},
		}}, "group_type runner requires group_id"),
		Entry("project_type without project_id", "val-nopid", RunnerSpec{Authentication: GitlabAuth{
			AccessToken:   &TokenSource{Value: "x"},
			CreateOptions: &RunnerCreateOptions{RunnerType: "project_type"},
		}}, "project_type runner requires project_id"),
		Entry("token value and secret_key_ref", "val-bothsrc", RunnerSpec{Authentication: GitlabAuth{
			Token: &TokenSource{Value: "x", SecretKeyRef: &SecretKeySelector{Name: "s"}},
		}}, "set either value or secret_key_ref"),
		Entry("secret_key_ref with empty name", "val-emptyref", RunnerSpec{Authentication: GitlabAuth{
			Token: &TokenSource{SecretKeyRef: &SecretKeySelector{Name: ""}},
		}}, "at least 1 chars long"),
		Entry("namespace_per_job set", "val-nsperjob", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{NamespacePerJob: true},
		}, "namespace_per_job is not supported"),
		Entry("namespace_overwrite_allowed set", "val-nsoverwrite", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{NamespaceOverwriteAllowed: ".*"},
		}, "namespace_overwrite_allowed is not supported"),
		// Two entries so that dropping any single source from the exclusivity list
		// fails a test; one pair alone leaves the third source unexercised.
		Entry("caCertificate value and secretKeyRef", "val-catwo", RunnerSpec{
			Authentication: valByoAuth(),
			CACertificate: &CASource{
				Value:        "-----BEGIN CERTIFICATE-----",
				SecretKeyRef: &CAKeyRef{Name: "ca-secret"},
			},
		}, "set only one of value, secretKeyRef, or configMapKeyRef"),
		// The runner rejects the whole config.toml if an NFS volume is missing a
		// required field, so admission has to catch an empty name too.
		Entry("nfs volume with an empty name", "val-nfsnoname", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{Volumes: &KubernetesVolumes{
				NFSVolumes: []KubernetesNFS{{
					Name: "", MountPath: "/mnt/nfs", Server: "10.0.0.1", Path: "/exports",
				}},
			}},
		}, "at least 1 chars long"),
		Entry("nfs volume with an empty server", "val-nfsnosrv", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{Volumes: &KubernetesVolumes{
				NFSVolumes: []KubernetesNFS{{
					Name: "nfs", MountPath: "/mnt/nfs", Server: "", Path: "/exports",
				}},
			}},
		}, "at least 1 chars long"),
		Entry("nfs volume with an empty path", "val-nfsnopath", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{Volumes: &KubernetesVolumes{
				NFSVolumes: []KubernetesNFS{{
					Name: "nfs", MountPath: "/mnt/nfs", Server: "10.0.0.1", Path: "",
				}},
			}},
		}, "at least 1 chars long"),
		Entry("nfs volume with an empty mount_path", "val-nfsnomp", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{Volumes: &KubernetesVolumes{
				NFSVolumes: []KubernetesNFS{{
					Name: "nfs", MountPath: "", Server: "10.0.0.1", Path: "/exports",
				}},
			}},
		}, "at least 1 chars long"),
		// The runner drops a profile it cannot use and only logs it, so a build
		// container would run unconfined while the spec claims otherwise.
		Entry("seccomp profile with an unknown type", "val-badseccomp", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{PodSecurityContext: &KubernetesPodSecurityContext{
				SeccompProfile: &KubernetesSeccompProfile{Type: "runtimedefault"},
			}},
		}, "Unsupported value"),
		Entry("seccomp Localhost without a profile path", "val-seccompnopath", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{PodSecurityContext: &KubernetesPodSecurityContext{
				SeccompProfile: &KubernetesSeccompProfile{Type: "Localhost"},
			}},
		}, "localhost_profile is required when type is Localhost"),
		// A typed client always sends type:"" because the field is a bare string,
		// so the enum is what rejects an unset type; required covers a raw object
		// that omits the key entirely.
		Entry("apparmor profile with no type at all", "val-apparmornotype", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{PodSecurityContext: &KubernetesPodSecurityContext{
				AppArmorProfile: &KubernetesAppArmorProfile{LocalhostProfile: "p"},
			}},
		}, `Unsupported value: ""`),
		Entry("apparmor Localhost without a profile name", "val-apparmornopath", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{BuildContainerSecurityContext: &KubernetesContainerSecurityContext{
				AppArmorProfile: &KubernetesAppArmorProfile{Type: "Localhost"},
			}},
		}, "localhost_profile is required when type is Localhost"),
		Entry("cleanup_resources_timeout with a bad unit", "val-badtimeout", RunnerSpec{
			Authentication: valByoAuth(),
			ExecutorConfig: KubernetesConfig{CleanupResourcesTimeout: "1d"},
		}, "should match"),
		Entry("caCertificate value and configMapKeyRef", "val-cacm", RunnerSpec{
			Authentication: valByoAuth(),
			CACertificate: &CASource{
				Value:           "-----BEGIN CERTIFICATE-----",
				ConfigMapKeyRef: &CAKeyRef{Name: "ca-cm"},
			},
		}, "set only one of value, secretKeyRef, or configMapKeyRef"),
	)

	It("rejects an explicitly empty inline token value (raw object)", func() {
		// A typed client drops value:"" via omitempty; only a raw object can carry
		// an explicit empty value, which the CEL size() guard must still reject.
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(GroupVersion.WithKind("Runner"))
		u.SetNamespace("default")
		u.SetName("val-emptyvalue")
		Expect(unstructured.SetNestedField(u.Object, "", "spec", "authentication", "token", "value")).To(Succeed())
		expectInvalid(k8sClient.Create(ctx, u), "set either value or secret_key_ref")
	})

	It("rejects a MultiRunner with no entries", func() {
		expectInvalid(k8sClient.Create(ctx, &MultiRunner{
			ObjectMeta: valMeta("val-mr-noentries"),
			Spec:       MultiRunnerSpec{Entries: []MultiRunnerEntry{}},
		}), "should have at least 1 items")
	})

	It("rejects a MultiRunner with duplicate entry names", func() {
		expectInvalid(k8sClient.Create(ctx, &MultiRunner{
			ObjectMeta: valMeta("val-mr-dup"),
			Spec: MultiRunnerSpec{Entries: []MultiRunnerEntry{
				{Name: "dup", Authentication: valByoAuth()},
				{Name: "dup", Authentication: valByoAuth()},
			}},
		}), "Duplicate value")
	})
})

// The manager placement fields (#83) are only useful if the generated CRD
// carries them. A misspelled json tag is pruned by the apiserver without an
// error, so these specs read the object back rather than trusting the create.
var _ = Describe("manager pod placement schema", func() {
	placement := func() RunnerSpec {
		return RunnerSpec{
			Authentication:     valByoAuth(),
			RunnerNodeSelector: map[string]string{"node-pool": "ci"},
			RunnerTolerations: []corev1.Toleration{{
				Key:      "dedicated",
				Operator: corev1.TolerationOpEqual,
				Value:    "ci",
				Effect:   corev1.TaintEffectNoSchedule,
			}},
			RunnerAffinity: &corev1.Affinity{
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
			RunnerPriorityClassName: "system-cluster-critical",
		}
	}

	It("stores every placement field on a Runner", func() {
		r := &Runner{ObjectMeta: valMeta("val-place-runner"), Spec: placement()}
		Expect(k8sClient.Create(ctx, r)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, r) })

		var stored Runner
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(r), &stored)).To(Succeed())
		Expect(stored.Spec.RunnerNodeSelector).To(HaveKeyWithValue("node-pool", "ci"))
		Expect(stored.Spec.RunnerTolerations).To(HaveLen(1))
		Expect(stored.Spec.RunnerTolerations[0].Effect).To(Equal(corev1.TaintEffectNoSchedule))
		Expect(stored.Spec.RunnerAffinity).NotTo(BeNil())
		Expect(stored.Spec.RunnerAffinity.NodeAffinity).NotTo(BeNil())
		Expect(stored.Spec.RunnerPriorityClassName).To(Equal("system-cluster-critical"))
	})

	It("stores every placement field on a MultiRunner", func() {
		spec := placement()
		m := &MultiRunner{ObjectMeta: valMeta("val-place-multi"), Spec: MultiRunnerSpec{
			Entries:                 []MultiRunnerEntry{{Name: "e1", Authentication: valByoAuth()}},
			RunnerNodeSelector:      spec.RunnerNodeSelector,
			RunnerTolerations:       spec.RunnerTolerations,
			RunnerAffinity:          spec.RunnerAffinity,
			RunnerPriorityClassName: spec.RunnerPriorityClassName,
		}}
		Expect(k8sClient.Create(ctx, m)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, m) })

		var stored MultiRunner
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(m), &stored)).To(Succeed())
		Expect(stored.Spec.RunnerNodeSelector).To(HaveKeyWithValue("node-pool", "ci"))
		Expect(stored.Spec.RunnerTolerations).To(HaveLen(1))
		Expect(stored.Spec.RunnerAffinity).NotTo(BeNil())
		Expect(stored.Spec.RunnerPriorityClassName).To(Equal("system-cluster-critical"))
	})

	// Positive control on the shape itself. corev1.Toleration carries no enum
	// markers upstream, so no value is rejected; what proves the field is a
	// typed object rather than a free-form map is that the structural schema
	// prunes a key it does not know.
	It("prunes an unknown key inside a toleration", func() {
		u := unstructuredRunner("val-place-prune", "runner_tolerations", []any{map[string]any{
			"key":      "dedicated",
			"operator": "Equal",
			"value":    "ci",
			"nonsense": "dropped",
		}})
		Expect(k8sClient.Create(ctx, u)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, u) })

		tolerations, found, err := unstructured.NestedSlice(u.Object, "spec", "runner_tolerations")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue(), "spec.runner_tolerations was pruned entirely")
		Expect(tolerations).To(HaveLen(1))
		Expect(tolerations[0]).To(HaveKeyWithValue("key", "dedicated"))
		Expect(tolerations[0]).NotTo(HaveKey("nonsense"))
	})
})
