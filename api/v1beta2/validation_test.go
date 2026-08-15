package v1beta2

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func valMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: "default"}
}

func valByoAuth() GitlabAuth {
	return GitlabAuth{Token: &TokenSource{Value: "glrt-x"}}
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

	DescribeTable("rejects an invalid Runner with a CEL message",
		func(name string, spec RunnerSpec, wantMsg string) {
			err := k8sClient.Create(ctx, &Runner{ObjectMeta: valMeta(name), Spec: spec})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(wantMsg))
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
		Entry("group_type without group_id", "val-nogid", RunnerSpec{Authentication: GitlabAuth{
			AccessToken:   &TokenSource{Value: "x"},
			CreateOptions: &RunnerCreateOptions{RunnerType: "group_type"},
		}}, "group_type runner requires group_id"),
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
	)

	It("rejects an explicitly empty inline token value (raw object)", func() {
		// A typed client drops value:"" via omitempty; only a raw object can carry
		// an explicit empty value, which the CEL size() guard must still reject.
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(GroupVersion.WithKind("Runner"))
		u.SetNamespace("default")
		u.SetName("val-emptyvalue")
		Expect(unstructured.SetNestedField(u.Object, "", "spec", "authentication", "token", "value")).To(Succeed())
		err := k8sClient.Create(ctx, u)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("set either value or secret_key_ref"))
	})

	It("rejects a MultiRunner with no entries", func() {
		err := k8sClient.Create(ctx, &MultiRunner{
			ObjectMeta: valMeta("val-mr-noentries"),
			Spec:       MultiRunnerSpec{Entries: []MultiRunnerEntry{}},
		})
		Expect(err).To(HaveOccurred())
	})

	It("rejects a MultiRunner with duplicate entry names", func() {
		err := k8sClient.Create(ctx, &MultiRunner{
			ObjectMeta: valMeta("val-mr-dup"),
			Spec: MultiRunnerSpec{Entries: []MultiRunnerEntry{
				{Name: "dup", Authentication: valByoAuth()},
				{Name: "dup", Authentication: valByoAuth()},
			}},
		})
		Expect(err).To(HaveOccurred())
	})
})
