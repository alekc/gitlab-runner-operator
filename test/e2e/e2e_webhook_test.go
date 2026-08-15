package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// expectRejected asserts the admission webhook rejects the create, and best
// effort deletes the object if it somehow slipped through.
func expectRejected(obj client.Object, because string) {
	GinkgoHelper()
	err := k8sClient.Create(context.Background(), obj)
	Expect(err).To(HaveOccurred(), because)
	if err == nil {
		_ = k8sClient.Delete(context.Background(), obj)
	}
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
		}, "both bring-your-own and managed auth must be rejected")
	})

	It("rejects a Runner with neither auth mode set", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-no-auth"),
			Spec:       gitlabv1beta2.RunnerSpec{GitlabInstanceURL: gitlabURL},
		}, "a Runner with no authentication must be rejected")
	})

	It("rejects namespace_per_job (dynamic build namespace)", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-nsperjob"),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication:    byoAuth(),
				ExecutorConfig:    gitlabv1beta2.KubernetesConfig{NamespacePerJob: true},
			},
		}, "namespace_per_job must be rejected")
	})

	It("rejects namespace_overwrite_allowed (non-static build namespace)", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-nsoverwrite"),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication:    byoAuth(),
				ExecutorConfig:    gitlabv1beta2.KubernetesConfig{NamespaceOverwriteAllowed: ".*"},
			},
		}, "namespace_overwrite_allowed must be rejected")
	})

	It("rejects a build namespace outside the runner's own (no allow-list)", func() {
		expectRejected(&gitlabv1beta2.Runner{
			ObjectMeta: objMeta("e2e-reject-crossns"),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication:    byoAuth(),
				ExecutorConfig:    gitlabv1beta2.KubernetesConfig{Namespace: "kube-system"},
			},
		}, "a cross-namespace executor namespace must be rejected by default")
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
		}, "duplicate MultiRunner entry names must be rejected")
	})

	It("rejects a MultiRunner with no entries", func() {
		expectRejected(&gitlabv1beta2.MultiRunner{
			ObjectMeta: objMeta("e2e-reject-no-entries"),
			Spec:       gitlabv1beta2.MultiRunnerSpec{GitlabInstanceURL: gitlabURL},
		}, "a MultiRunner with no entries must be rejected")
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
		_ = k8sClient.Delete(context.Background(), obj)
	})
})
