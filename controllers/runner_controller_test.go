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
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
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
		RunnerNamespace = "default"
		interval        = time.Millisecond * 250
	)
	var timeout = time.Second * 10

	// if we are running a debugger, then increase timeout to 10 minutes to prevent killed debug sessions
	if val, ok := os.LookupEnv("TEST_TIMEOUT"); ok {
		newTimeout, err := strconv.Atoi(val)
		if err == nil {
			timeout = time.Second * time.Duration(newTimeout)
		}
	}

	Context("when creating a new runner", func() {
		runnerName := fmt.Sprintf("test-runner-%s", uuid.New().String()[0:6])
		ctx := context.Background()
		runner := &v1beta1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name: runnerName,
			},
		}
		namespacedDependencyName := types.NamespacedName{Name: runner.ChildName(), Namespace: RunnerNamespace}
		runnerObjectKey := types.NamespacedName{Name: runnerName, Namespace: RunnerNamespace}
		It("should create a runner object", func() {
			newRunner := &v1beta1.Runner{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta1.GroupVersion.String(),
					Kind:       "Runner",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      runnerName,
					Namespace: RunnerNamespace,
				},
				Spec: v1beta1.RunnerSpec{
					RegistrationConfig: v1beta1.RegisterNewRunnerOptions{
						Token:   pointer.StringPtr("zTS6g2Q8bp8y13_ynfpN"),
						TagList: []string{"testing-runner-operator"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, newRunner)).Should(Succeed())

			// fetch the runner crd entity
			Eventually(func() bool {
				return k8sClient.Get(ctx, runnerObjectKey, runner) == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("should create required rbac authorizations", func() {
			Expect(runner.UID).NotTo(BeEmpty())

			// there should be required rbac created
			// sa first
			sa := corev1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, namespacedDependencyName, &sa) == nil
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
			Expect(role.Rules[0].Resources).To(BeEquivalentTo([]string{"pods", "pods/exec", "pods/attach", "secrets", "configmaps"}))

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

		It("should have updated the runner status with the authorization token", func() {
			newRunner := &v1beta1.Runner{}
			// fetch the runner crd entity
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, runnerObjectKey, newRunner); err != nil {
					return false
				}
				return newRunner.Status.AuthenticationToken != ""
			}, timeout, interval).Should(BeTrue())
			Expect(newRunner.Status.Error).To(BeEmpty())
			Expect(newRunner.Status.AuthenticationToken).To(BeEquivalentTo("fc54da1422d38c11f4727c6c32f67738"))
		})

		var configMap corev1.ConfigMap
		It("should have generated config map with gitlab runner config", func() {
			Eventually(func() bool {
				return k8sClient.Get(ctx, namespacedDependencyName, &configMap) == nil
			}, timeout, interval).Should(BeTrue())
			// validate that a proper key has been defined
			Expect(configMap.Data).Should(HaveKey(configMapKeyName))
		})

		It("should have set the config map hashkey to the runner status", func() {
			Eventually(func() string {
				_ = k8sClient.Get(ctx, runnerObjectKey, runner)
				return runner.Status.ConfigMapVersion
			}, timeout, interval).Should(Equal(configMap.Annotations[configVersionAnnotationKey]))
		})
		It("should have generated deployment", func() {
			var deployment appsv1.Deployment
			Eventually(func() bool {
				return k8sClient.Get(ctx, namespacedDependencyName, &deployment) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(deployment.OwnerReferences).NotTo(BeEmpty())
			Expect(deployment.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
			Expect(deployment.Annotations).To(HaveKey(configVersionAnnotationKey))
			Expect(deployment.Annotations[configVersionAnnotationKey]).To(BeEquivalentTo(configMap.Annotations[configVersionAnnotationKey]))
		})
	})

	Context("Test the changes in the configuration", func() {
		var runner v1beta1.Runner
		runnerName := fmt.Sprintf("test-runner-change-%s", uuid.New().String()[0:6])

		ctx := context.Background()
		var namespacedDependencyName types.NamespacedName

		// create a new runner
		It("Should create a new runner", func() {
			// create a runner
			Expect(k8sClient.Create(ctx, &v1beta1.Runner{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      runnerName,
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
				return k8sClient.Get(ctx, types.NamespacedName{Name: runnerName, Namespace: RunnerNamespace},
					&runner) == nil
			}, timeout, interval).Should(BeTrue())
			namespacedDependencyName = types.NamespacedName{Name: runner.ChildName(), Namespace: RunnerNamespace}
		})

		var configMap corev1.ConfigMap
		var oldConfigMapVersion string
		It("should have a valid config map created", func() {
			// obtain latest config map version
			Eventually(func() bool {
				return k8sClient.Get(ctx, namespacedDependencyName, &configMap) == nil
			}, timeout, interval).Should(BeTrue())
			oldConfigMapVersion = configMap.Annotations[configVersionAnnotationKey]
		})

		var dp appsv1.Deployment
		It("should have created a dp", func() {
			// find related deployment.
			// it should have the same config map version
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedDependencyName, &dp)
				if err != nil {
					return false
				}
				return dp.Annotations[configVersionAnnotationKey] == oldConfigMapVersion &&
					dp.Spec.Template.Annotations[configVersionAnnotationKey] == oldConfigMapVersion
			}, timeout, interval).Should(BeTrue())
		})

		It("Runner should have the config map version as cm", func() {
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerName, Namespace: RunnerNamespace},
					&runner); err != nil {
					return false
				}
				return runner.Status.ConfigMapVersion == oldConfigMapVersion
			}, timeout, interval).Should(BeTrue())
		})

		It("Should update both configmap and deployment when the spec is changed", func() {
			// verify that deployment has an expected
			dpCheckFunc := func(desiredConfigMapVersion string) bool {
				var dp appsv1.Deployment
				if err := k8sClient.Get(ctx, namespacedDependencyName, &dp); err != nil {
					return false
				}
				return dp.Annotations[configVersionAnnotationKey] == desiredConfigMapVersion
			}
			Eventually(dpCheckFunc(oldConfigMapVersion), timeout, interval).Should(BeTrue())

			// update the runner spec forcing change in final configuration
			runner.Spec.Concurrent = 2
			Expect(k8sClient.Update(ctx, &runner)).To(Succeed())

			// runner should get a new hash version
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerName, Namespace: RunnerNamespace},
					&runner)
				return err == nil && runner.Status.ConfigMapVersion != oldConfigMapVersion
			}, timeout, interval).Should(BeTrue())

			// wait until the configmap is updated and fetch the new version
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedDependencyName, &configMap)
				if err != nil {
					return false
				}
				return configMap.Annotations[configVersionAnnotationKey] == runner.Status.ConfigMapVersion
			}, timeout, interval).Should(BeTrue())

			// verify that our deployment has been amended with a new version
			Eventually(dpCheckFunc(runner.Status.ConfigMapVersion), timeout, interval).Should(BeTrue())
		})

		It("Should re register when tags are changed", func() {
			oldConfigMapVersion = runner.Status.ConfigMapVersion

			// update the runner spec forcing change in final configuration
			runner.Spec.RegistrationConfig.TagList = []string{"new", "tag", "list"}
			Expect(k8sClient.Update(ctx, &runner)).To(Succeed())

			// runner should get a new hash version
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: runnerName, Namespace: RunnerNamespace},
					&runner)
				return err == nil && runner.Status.ConfigMapVersion != oldConfigMapVersion
			}, timeout, interval).Should(BeTrue())

			// wait until the configmap is updated and fetch the new version
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedDependencyName, &configMap)
				if err != nil {
					return false
				}
				return configMap.Annotations[configVersionAnnotationKey] == runner.Status.ConfigMapVersion
			}, timeout, interval).Should(BeTrue())

			// verify that our deployment has been amended with a new version
			Eventually(func() bool {
				var dp appsv1.Deployment
				if err := k8sClient.Get(ctx, namespacedDependencyName, &dp); err != nil {
					return false
				}
				return dp.Annotations[configVersionAnnotationKey] == runner.Status.ConfigMapVersion
			}).Should(BeTrue())
		})
	})
})
