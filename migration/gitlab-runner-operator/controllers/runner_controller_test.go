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
	"gitlab.k8s.alekc.dev/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
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
		runner := &v1beta1.Runner{
			ObjectMeta: metav1.ObjectMeta{Name: RunnerName},
		}
		namespacedDependencyName := types.NamespacedName{Name: runner.ChildName(), Namespace: RunnerNamespace}

		It("should create a runner object", func() {
			newRunner := &v1beta1.Runner{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta1.GroupVersion.String(),
					Kind:       "Runner",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      RunnerName,
					Namespace: RunnerNamespace,
				},
				Spec: v1beta1.RunnerSpec{
					RegistrationConfig: v1beta1.RegisterNewRunnerOptions{
						Token:   pointer.StringPtr("rYwg6EogqxSuvsFCVvAT"),
						TagList: []string{"testing-runner-operator"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, newRunner)).Should(Succeed())

			// fetch the runner crd entity
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}, runner) == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("should create required rbac authorizations", func() {
			By("By creating a new CronJob")

			// there should be required rbac created
			// sa first
			sa := corev1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(
					ctx,
					namespacedDependencyName,
					&sa,
				) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(sa.OwnerReferences).NotTo(BeEmpty())
			Expect(sa.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))

			// fetch the role and validate it's values
			var role v1.Role
			Eventually(func() bool {
				return k8sClient.Get(ctx, namespacedDependencyName, &role) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(role.OwnerReferences).NotTo(BeEmpty())
			Expect(role.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
			Expect(role.Rules).NotTo(BeEmpty())
			Expect(role.Rules[0].APIGroups).To(BeEquivalentTo([]string{"*"}))
			Expect(role.Rules[0].Verbs).To(BeEquivalentTo([]string{"get", "list", "watch", "create", "patch", "delete"}))
			Expect(role.Rules[0].Resources).To(BeEquivalentTo([]string{"pods", "pods/exec", "secrets"}))

			// and finally, check the actual role binding
			var roleBinding v1.RoleBinding
			Eventually(func() bool {
				return k8sClient.Get(ctx, namespacedDependencyName, &roleBinding) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(roleBinding.OwnerReferences).NotTo(BeEmpty())
			Expect(roleBinding.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
			Expect(roleBinding.Subjects).NotTo(BeEmpty())
			Expect(roleBinding.Subjects[0]).To(BeEquivalentTo(v1.Subject{
				Kind:      "ServiceAccount",
				Name:      namespacedDependencyName.Name,
				Namespace: namespacedDependencyName.Namespace,
			}))
			Expect(roleBinding.RoleRef).To(BeEquivalentTo(v1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     namespacedDependencyName.Name,
			}))
		})

		It("should have updated the runner status with the token", func() {
			newRunner := &v1beta1.Runner{}
			// fetch the runner crd entity
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}, newRunner) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(newRunner.Status.Error).To(BeEmpty())
			Expect(newRunner.Status.AuthenticationToken).To(BeEquivalentTo("xyz"))
		})

		It("should have generated config map with gitlab runner config", func() {
			var configMap corev1.ConfigMap
			Eventually(func() bool {
				return k8sClient.Get(ctx, namespacedDependencyName, &configMap) == nil
			}, timeout, interval).Should(BeTrue())
			// validate that a proper key has been defined
			Expect(configMap.Data).Should(HaveKey(configMapKeyName))
		})

		It("should authenticate against the gitlab server and obtain the auth token", func() {
			Eventually(func() bool {
				var newRunner v1beta1.Runner
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: runner.Namespace,
						Name:      runner.Name,
					},
					&newRunner,
				)
				if err == nil && newRunner.Status.AuthenticationToken != "" {
					// since we are here, update the runner with a fresher option
					runner = &newRunner
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
			// todo: check for annotations as well
		})
		It("should have generated deployment", func() {
			var deployment appsv1.Deployment
			Eventually(func() bool {
				return k8sClient.Get(ctx, namespacedDependencyName, &deployment) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(deployment.OwnerReferences).NotTo(BeEmpty())
			Expect(deployment.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
			Expect(deployment.Annotations).To(HaveKey(configVersionAnnotationKey))
		})
	})

	Context("Test the changes in the configuration", func() {
		var runner v1beta1.Runner
		var namespacedDependencyName types.NamespacedName
		var configMapVersion string

		ctx := context.Background()

		It("Should create a new runner", func() {
			// create a runner
			Expect(k8sClient.Create(ctx, &v1beta1.Runner{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner-config-changes",
					Namespace: RunnerNamespace,
				},
				Spec: v1beta1.RunnerSpec{
					RegistrationConfig: v1beta1.RegisterNewRunnerOptions{
						Token:   pointer.StringPtr("rYwg6EogqxSuvsFCVvAT"),
						TagList: []string{"testing-runner-operator"},
					},
				},
			})).Should(Succeed())

			// fetch latest runner version from the cluster

			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}, &runner) == nil
			}, timeout, interval).Should(BeTrue())
			namespacedDependencyName = types.NamespacedName{Name: runner.ChildName(), Namespace: RunnerNamespace}
		})
		It("should have valid config map created", func() {
			// obtain latest config map version
			Eventually(func() bool {
				var configMap corev1.ConfigMap
				err := k8sClient.Get(ctx, namespacedDependencyName, &configMap)
				if err != nil {
					return false
				}
				Expect(configMap.Annotations).To(HaveKey(configVersionAnnotationKey))
				configMapVersion = configMap.Annotations[configVersionAnnotationKey]
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(configMapVersion).NotTo(BeEmpty())
		})

		It("Should update both configmap and deployment when the spec is changed", func() {

			// verify that deployment has an expected
			dpCheck := func(desiredConfigMapVersion string) bool {
				var dp appsv1.Deployment
				err := k8sClient.Get(ctx, namespacedDependencyName, &dp)
				if err != nil {
					return false
				}
				// validate the configmap
				if dp.Annotations == nil {
					return false
				}
				if val, ok := dp.Annotations[configVersionAnnotationKey]; ok && val == desiredConfigMapVersion {
					return true
				}
				return false
			}
			Eventually(dpCheck(configMapVersion), timeout, interval).Should(BeTrue())

			// update the runner spec forcing change in final configuration
			runner.Spec.Concurrent = 2
			Expect(k8sClient.Update(ctx, &runner)).To(Succeed())

			// wait until the configmap is updated and fetch the new version
			Eventually(func() bool {
				var configMap corev1.ConfigMap
				err := k8sClient.Get(ctx, namespacedDependencyName, &configMap)
				if err != nil {
					return false
				}

				Expect(configMap.Annotations).To(HaveKey(configVersionAnnotationKey))
				if val, ok := configMap.Annotations[configVersionAnnotationKey]; ok && val != configMapVersion {
					configMapVersion = val
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
			Expect(configMapVersion).NotTo(BeEmpty())

			// verify that our deployment has been amended with a new version
			Eventually(dpCheck(configMapVersion), timeout, interval).Should(BeTrue())
		})
	})
})
