package v1beta2

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func valMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: "default"}
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

	causes := status.Status().Details.Causes
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
