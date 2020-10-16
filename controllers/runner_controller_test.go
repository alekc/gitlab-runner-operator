/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// +kubebuilder:docs-gen:collapse=Apache License

/*
As usual, we start with the necessary imports. We also define some utility variables.
*/
package controllers

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gitlabRunOp "go.alekc.dev/gitlab-runner-operator/api/v1alpha1"
	"go.alekc.dev/gitlab-runner-operator/internal/generate"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = Describe("Runner controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		RunnerName      = "test-runner"
		RunnerNamespace = "default"

		duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	var timeout = time.Second * 10
	// if we are running a debugger, then increase timeout to 10 minutes to prevent killed debug sessions
	if _, ok := os.LookupEnv("DebuggerRunning"); ok {
		timeout = time.Minute * 10
	}

	Context("When creating a runner type crd", func() {
		ctx := context.Background()
		runner := &gitlabRunOp.Runner{}
		It("should create a runner instance without any error", func() {
			cronJob := &gitlabRunOp.Runner{
				TypeMeta: metav1.TypeMeta{
					APIVersion: gitlabRunOp.GroupVersion.String(),
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      RunnerName,
					Namespace: RunnerNamespace,
				},
				Spec: gitlabRunOp.RunnerSpec{
					RegistrationConfig: gitlabRunOp.RegisterNewRunnerOptions{
						Token:   pointer.StringPtr("12345"),
						TagList: []string{"testing-runner-operator"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cronJob)).Should(Succeed())

			// fetch the runner crd entity
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}, runner) == nil
			}, timeout, interval).Should(BeTrue())
		})

		//
		It("should create required rbac authorizations", func() {
			By("By creating a new CronJob")

			// there should be required rbac created
			rbacName := types.NamespacedName{Name: generate.RbacName(runner), Namespace: RunnerNamespace}

			// sa first
			sa := corev1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(
					ctx,
					rbacName,
					&sa,
				) == nil
			}).Should(BeTrue())
			Expect(sa.OwnerReferences).NotTo(BeEmpty())
			Expect(sa.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))

			// fetch the role and validate it's values
			var role v1.Role
			Eventually(func() bool {
				return k8sClient.Get(ctx, rbacName, &role) == nil
			}).Should(BeTrue())
			Expect(role.OwnerReferences).NotTo(BeEmpty())
			Expect(role.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
			Expect(role.Rules).NotTo(BeEmpty())
			Expect(role.Rules[0].APIGroups).To(BeEquivalentTo([]string{"*"}))
			Expect(role.Rules[0].Verbs).To(BeEquivalentTo([]string{"get", "list", "watch", "create", "patch", "delete"}))
			Expect(role.Rules[0].Resources).To(BeEquivalentTo([]string{"pods", "pods/exec", "secrets"}))

			// and finally, check the actual role binding
			var roleBinding v1.RoleBinding
			Eventually(func() bool {
				return k8sClient.Get(ctx, rbacName, &roleBinding) == nil
			}).Should(BeTrue())
			Expect(roleBinding.OwnerReferences).NotTo(BeEmpty())
			Expect(roleBinding.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
			Expect(roleBinding.Subjects).NotTo(BeEmpty())
			Expect(roleBinding.Subjects[0]).To(BeEquivalentTo(v1.Subject{
				Kind:      "ServiceAccount",
				Name:      rbacName.Name,
				Namespace: rbacName.Namespace,
			}))
			Expect(roleBinding.RoleRef).To(BeEquivalentTo(v1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     rbacName.Name,
			}))
		})
	})
})
