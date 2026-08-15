package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// expectRejected asserts an admission-webhook denial whose message contains
// wantMsg, so each case maps to its own reason and a stray CRD, RBAC, or
// apiserver error cannot pass. Cleanup is registered first: if the webhook
// wrongly admits, the assertion aborts the spec, so this is the only delete.
func expectRejected(obj client.Object, wantMsg string) {
	GinkgoHelper()
	DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), obj) })
	err := k8sClient.Create(context.Background(), obj)
	Expect(err).To(HaveOccurred(), "expected the webhook to reject: %s", wantMsg)
	Expect(err.Error()).To(ContainSubstring("denied the request"),
		"want an admission-webhook denial, got: %v", err)
	Expect(err.Error()).To(ContainSubstring(wantMsg))
}

func byoAuth() gitlabv1beta2.GitlabAuth {
	return gitlabv1beta2.GitlabAuth{Token: &gitlabv1beta2.TokenSource{Value: "glrt-placeholder"}}
}

func managedAuth() gitlabv1beta2.GitlabAuth {
	return gitlabv1beta2.GitlabAuth{
		AccessToken:   &gitlabv1beta2.TokenSource{Value: "glpat-placeholder"},
		CreateOptions: managedCreateOptions([]string{jobTag}),
	}
}

var _ = Describe("Admission webhook validation", func() {
	It("rejects a Runner with both auth modes set", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-both-auth"),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication: gitlabv1beta2.GitlabAuth{
					Token:         &gitlabv1beta2.TokenSource{Value: "glrt-placeholder"},
					AccessToken:   &gitlabv1beta2.TokenSource{Value: "glpat-placeholder"},
					CreateOptions: managedCreateOptions([]string{jobTag}),
				},
			},
		}, "set either a pre-created authentication token or create_options, not both")
	})

	It("rejects a Runner with neither auth mode set", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-no-auth"),
			Spec:       gitlabv1beta2.RunnerSpec{GitlabInstanceURL: gitlabURL},
		}, "one of token or create_options must be set")
	})

	It("rejects namespace_per_job (dynamic build namespace)", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-nsperjob"),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication:    byoAuth(),
				ExecutorConfig:    gitlabv1beta2.KubernetesConfig{NamespacePerJob: true},
			},
		}, "namespace_per_job is not supported")
	})

	It("rejects namespace_overwrite_allowed (non-static build namespace)", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-nsoverwrite"),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication:    byoAuth(),
				ExecutorConfig:    gitlabv1beta2.KubernetesConfig{NamespaceOverwriteAllowed: ".*"},
			},
		}, "namespace_overwrite_allowed is not supported")
	})

	It("rejects a build namespace outside the runner's own (no allow-list)", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-crossns"),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication:    byoAuth(),
				ExecutorConfig:    gitlabv1beta2.KubernetesConfig{Namespace: "kube-system"},
			},
		}, "is not permitted")
	})

	It("rejects a MultiRunner with duplicate entry names", func() {
		expectRejected(&gitlabv1beta2.MultiRunner{
			ObjectMeta: objMeta("e2e-reject-dup-entries"),
			Spec: gitlabv1beta2.MultiRunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Entries: []gitlabv1beta2.MultiRunnerEntry{
					{Name: "dup", Authentication: byoAuth()},
					{Name: "dup", Authentication: managedAuth()},
				},
			},
		}, "duplicate entry name")
	})

	It("rejects a MultiRunner with no entries", func() {
		// Send an explicit empty slice (entries: []): it satisfies the CRD's
		// required-array schema so the webhook is what rejects it. A nil slice
		// serializes to null and would fail as a 422 schema error instead.
		expectRejected(&gitlabv1beta2.MultiRunner{
			ObjectMeta: objMeta("e2e-reject-no-entries"),
			Spec: gitlabv1beta2.MultiRunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Entries:           []gitlabv1beta2.MultiRunnerEntry{},
			},
		}, "a multirunner requires at least one entry")
	})

	It("accepts a valid bring-your-own Runner (positive control)", func() {
		obj := &gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-accept-byo-control"),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication:    byoAuth(),
			},
		}
		Expect(k8sClient.Create(context.Background(), obj)).To(Succeed(), "a valid Runner must be accepted")
		DeferCleanup(func() {
			Expect(k8sClient.Delete(context.Background(), obj)).To(Succeed())
		})
	})
})
