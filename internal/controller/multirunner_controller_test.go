package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/generate"
	internalTypes "gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MultiRunner controller", func() {
	var ns string
	var name string

	BeforeEach(func() {
		var err error
		ns, err = CreateNamespace(k8sClient)
		Expect(err).ToNot(HaveOccurred())
		name = fmt.Sprintf("test-mrunner-%s", generate.RandomString(5))
	})

	AfterEach(func() {
		_ = k8sClient.Delete(context.Background(), &v1beta2.MultiRunner{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground), client.GracePeriodSeconds(0))
		_ = k8sClient.Delete(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground), client.GracePeriodSeconds(0))
	})

	It("registers a managed entry and a bring-your-own-token entry", func() {
		ctx := context.Background()
		mr := &v1beta2.MultiRunner{
			TypeMeta:   metav1.TypeMeta{APIVersion: v1beta2.GroupVersion.String(), Kind: "MultiRunner"},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: v1beta2.MultiRunnerSpec{
				GitlabInstanceURL: "https://gitlab.com/",
				Entries: []v1beta2.MultiRunnerEntry{
					{
						Name: "byo",
						Authentication: v1beta2.GitlabAuth{
							Token: &v1beta2.TokenSource{Value: "glrt-byo-token"},
						},
					},
					{
						Name: "managed",
						Authentication: v1beta2.GitlabAuth{
							AccessToken: &v1beta2.TokenSource{Value: "glpat-access"},
							CreateOptions: &v1beta2.RunnerCreateOptions{
								RunnerType:  "instance_type",
								RunUntagged: ptr.To(true),
								TagList:     []string{"managed-tag"},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, mr)).To(Succeed())

		key := types.NamespacedName{Name: name, Namespace: ns}
		created := &v1beta2.MultiRunner{}
		By("waiting for the runner to become ready")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, created); err != nil {
				return false
			}
			return created.Status.Ready
		}, timeout, interval).Should(BeTrue())

		By("recording the managed entry's GitLab id and registration hash per entry name")
		Expect(created.Status.RunnerIDs["managed"]).To(Equal(2))
		Expect(created.Status.RegistrationHashes["managed"]).ToNot(BeEmpty())

		By("rendering config into a Secret with per-entry token keys (never a ConfigMap)")
		var secret corev1.Secret
		secretKey := types.NamespacedName{Name: created.ChildName(), Namespace: ns}
		Eventually(func() bool {
			return k8sClient.Get(ctx, secretKey, &secret) == nil
		}, timeout, interval).Should(BeTrue())
		Expect(secret.Type).To(Equal(corev1.SecretTypeOpaque))
		Expect(secret.Data).To(HaveKey(internalTypes.ConfigMapKeyName))
		Expect(secret.Data).To(HaveKey(internalTypes.ConfigTokenKeyPrefix + "byo"))
		Expect(secret.Data).To(HaveKey(internalTypes.ConfigTokenKeyPrefix + "managed"))
		Expect(string(secret.Data[internalTypes.ConfigTokenKeyPrefix+"byo"])).To(Equal("glrt-byo-token"))
	})
})
