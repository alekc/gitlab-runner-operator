package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

const (
	e2eNamespace = "default"
	timeout      = 4 * time.Minute
	interval     = 5 * time.Second
)

var _ = Describe("Managed Runner against live GitLab", func() {
	It("registers on GitLab, becomes Ready, brings up the manager, and cleans up on delete", func() {
		ctx := context.Background()
		name := fmt.Sprintf("e2e-managed-%d", time.Now().Unix())
		key := types.NamespacedName{Name: name, Namespace: e2eNamespace}

		runner := &gitlabv1beta2.Runner{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: e2eNamespace},
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication: gitlabv1beta2.GitlabAuth{
					AccessToken: &gitlabv1beta2.TokenSource{Value: gitlabToken},
					CreateOptions: &gitlabv1beta2.RunnerCreateOptions{
						RunnerType:  "project_type",
						ProjectID:   ptr.To(projectID),
						RunUntagged: ptr.To(true),
						TagList:     []string{"e2e"},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
		DeferCleanup(func() {
			_ = k8sClient.Delete(context.Background(), &gitlabv1beta2.Runner{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: e2eNamespace},
			})
		})

		By("becoming Ready with a GitLab-assigned runner id (proves the create call reached GitLab)")
		var got gitlabv1beta2.Runner
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, key, &got)).To(Succeed())
			g.Expect(got.Status.Ready).To(BeTrue())
			g.Expect(got.Status.RunnerID).NotTo(BeZero())
		}, timeout, interval).Should(Succeed())

		By("bringing up the runner manager Deployment")
		Eventually(func(g Gomega) {
			var dep appsv1.Deployment
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: got.ChildName(), Namespace: e2eNamespace}, &dep)).To(Succeed())
			g.Expect(dep.Status.AvailableReplicas).To(BeNumerically(">=", 1))
		}, timeout, interval).Should(Succeed())

		By("deleting the Runner and confirming the finalizer completes (GitLab-side cleanup)")
		Expect(k8sClient.Delete(ctx, &got)).To(Succeed())
		Eventually(func() bool {
			return apierrors.IsNotFound(k8sClient.Get(ctx, key, &gitlabv1beta2.Runner{}))
		}, timeout, interval).Should(BeTrue())
	})
})

var _ = Describe("Admission webhook", func() {
	It("rejects an unsupported dynamic build namespace (namespace_per_job)", func() {
		ctx := context.Background()
		bad := &gitlabv1beta2.Runner{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-reject-nsperjob", Namespace: e2eNamespace},
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication: gitlabv1beta2.GitlabAuth{
					Token: &gitlabv1beta2.TokenSource{Value: "placeholder"},
				},
				ExecutorConfig: gitlabv1beta2.KubernetesConfig{NamespacePerJob: true},
			},
		}
		err := k8sClient.Create(ctx, bad)
		Expect(err).To(HaveOccurred(), "the validating webhook should reject namespace_per_job")
		if err == nil {
			_ = k8sClient.Delete(ctx, bad)
		}
	})
})
