package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// expectRejected asserts the apiserver rejects the create as Invalid (a CRD
// schema or CEL rule) with a message containing wantMsg. Cleanup is registered
// first so a wrongly-admitted object cannot leak when the assertion aborts.
func expectRejected(obj client.Object, wantMsg string) {
	GinkgoHelper()
	DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), obj) })
	err := k8sClient.Create(context.Background(), obj)
	Expect(err).To(HaveOccurred(), "expected the apiserver to reject: %s", wantMsg)
	Expect(apierrors.IsInvalid(err)).To(BeTrue(), "want an Invalid (schema/CEL) rejection, got: %v", err)
	Expect(err.Error()).To(ContainSubstring(wantMsg))
}

func byoAuth() gitlabv1beta2.GitlabAuth {
	return gitlabv1beta2.GitlabAuth{Token: &gitlabv1beta2.TokenSource{Value: "glrt-placeholder"}}
}

// The admission webhook was removed: static rules are CRD schema + CEL, enforced
// by the apiserver, and the executor-namespace allow-list is enforced by the
// reconciler (a bad namespace goes NotReady instead of being rejected).
var _ = Describe("CRD validation", func() {
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

	It("rejects a MultiRunner with duplicate entry names", func() {
		expectRejected(&gitlabv1beta2.MultiRunner{
			ObjectMeta: objMeta("e2e-reject-dup-entries"),
			Spec: gitlabv1beta2.MultiRunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Entries: []gitlabv1beta2.MultiRunnerEntry{
					{Name: "dup", Authentication: byoAuth()},
					{Name: "dup", Authentication: byoAuth()},
				},
			},
		}, "Duplicate value")
	})

	It("rejects a MultiRunner with no entries", func() {
		expectRejected(&gitlabv1beta2.MultiRunner{
			ObjectMeta: objMeta("e2e-reject-no-entries"),
			Spec: gitlabv1beta2.MultiRunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Entries:           []gitlabv1beta2.MultiRunnerEntry{},
			},
		}, "should have at least 1 items")
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

var _ = Describe("Reconciler executor-namespace enforcement", func() {
	It("marks a runner NotReady when its executor namespace is not permitted", func() {
		ctx := context.Background()
		name := uniqueName("e2e-badns")
		runner := &gitlabv1beta2.Runner{
			ObjectMeta: objMeta(name),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication:    byoAuth(),
				// kube-system is neither the runner's own namespace nor in the
				// operator's allow-list, so the reconciler must refuse it.
				ExecutorConfig: gitlabv1beta2.KubernetesConfig{Namespace: "kube-system"},
			},
		}
		By("admission accepting it (the check moved from the webhook to the reconciler)")
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
		DeferCleanup(func() { deleteRunnerCR(name) })

		By("the reconciler reporting NotReady with a namespace error")
		Eventually(func(g Gomega) {
			var got gitlabv1beta2.Runner
			g.Expect(k8sClient.Get(ctx, key(name), &got)).To(Succeed())
			g.Expect(got.Status.Ready).To(BeFalse())
			g.Expect(got.Status.Error).To(ContainSubstring("is not permitted"))
		}, timeout, interval).Should(Succeed())
	})
})
