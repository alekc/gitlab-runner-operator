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
package controller

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	v1 "k8s.io/api/rbac/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/generate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:docs-gen:collapse=Imports

type testCase struct {
	Runner         *v1beta2.Runner
	CheckCondition func(*v1beta2.Runner) bool
	CheckRunner    func(*v1beta2.Runner)
}

type testCaseTweak func(*testCase)

const (
	interval = time.Millisecond * 250
)

var timeout = time.Second * 10

var _ = Describe("Runner controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	var RunnerNamespace string
	var RunnerName string

	// if we are running a debugger, then increase timeout to 10 minutes to prevent killed debug sessions
	if val, ok := os.LookupEnv("TEST_TIMEOUT"); ok {
		newTimeout, err := strconv.Atoi(val)
		if err == nil {
			timeout = time.Second * time.Duration(newTimeout)
		}
	}

	// before every test
	BeforeEach(func() {
		var err error
		// create a runner namespace
		RunnerNamespace, err = CreateNamespace(k8sClient)
		Expect(err).ToNot(HaveOccurred())

		// generate a random name for the runner
		RunnerName = fmt.Sprintf("test-runner-%s", generate.RandomString(5))
	})

	// after each test perform some cleaning actions
	AfterEach(func() {
		// delete the runner
		Expect(k8sClient.Delete(context.Background(), &v1beta2.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RunnerName,
				Namespace: RunnerNamespace,
			},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())

		// delete namespace
		Expect(k8sClient.Delete(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: RunnerNamespace,
			},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
	})

	DescribeTable(
		"When reconciling Gitlab Runner",
		func(tweaks ...testCaseTweak) {
			tc := &testCase{
				Runner: defaultRunner(RunnerName, RunnerNamespace),
				CheckCondition: func(runner *v1beta2.Runner) bool {
					return runner.UID != "" && runner.Status.Ready
				},
			}
			for _, tweak := range tweaks {
				tweak(tc)
			}

			ctx := context.Background()
			By("creating a runner")
			Expect(Expect(k8sClient.Create(ctx, tc.Runner)).To(Succeed()))

			//
			esKey := types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}
			createdRunner := &v1beta2.Runner{}
			By("checking the runner condition")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, esKey, createdRunner)
				if err != nil {
					return false
				}
				return tc.CheckCondition(createdRunner)
			}, timeout, interval).Should(BeTrue())

			//
			tc.CheckRunner(createdRunner)
		},
		Entry("Should support setting of env var for build env", caseEnvironmentIsSpecified),
		Entry("Should have created a different registration on tag update", caseTagsChanged),
		Entry("Should have created a different registration on registration token update", caseRegistrationTokenChanged),
		Entry("Should have updated runner status with auth token", caseTestAuthToken),
		Entry("Should have created required RBAC", caseRBACCheck),
		Entry("Should have generated config map", caseGeneratedConfigMap),
		Entry("Should have generated deployment", caseCheckDeployment),
		Entry("On spec change, config map should be updated", caseSpecChanged),
	)
})

func caseRBACCheck(tc *testCase) {
	tc.CheckRunner = func(runner *v1beta2.Runner) {
		ctx := context.TODO()

		// service  account should be created
		var sa corev1.ServiceAccount
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nameSpacedDependencyName(runner), &sa)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		// fetch created role
		var role v1.Role
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nameSpacedDependencyName(runner), &role)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		var roleBinding v1.RoleBinding
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nameSpacedDependencyName(runner), &roleBinding)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}
}

// caseTestAuthToken checks that a managed runner records the token and id that
// GitLab (the mock) returned.
func caseTestAuthToken(tc *testCase) {
	tc.CheckRunner = func(runner *v1beta2.Runner) {
		Expect(runner.Status.Error).To(BeEmpty())
		Expect(runner.Status.AuthenticationToken).ToNot(BeEmpty())
		Expect(runner.Status.RunnerID).To(Equal(1))
	}
}

// caseGeneratedConfigMap checks if a new config map is generated
func caseGeneratedConfigMap(tc *testCase) {
	ctx := context.Background()

	tc.CheckRunner = func(runner *v1beta2.Runner) {
		var configMap corev1.ConfigMap
		Eventually(func() bool {
			return k8sClient.Get(ctx, nameSpacedDependencyName(runner), &configMap) == nil
		}, timeout, interval).Should(BeTrue())

		Expect(configMap.OwnerReferences).NotTo(BeEmpty())
		Expect(configMap.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
		Expect(configMap.Data).Should(HaveKey(configMapKeyName), "Child config map should have %s data entry", configMapKeyName)
		Expect(runner.Status.ConfigMapVersion).Should(Not(BeEmpty()))
	}
}

func caseCheckDeployment(tc *testCase) {
	ctx := context.Background()
	tc.CheckRunner = func(runner *v1beta2.Runner) {
		var deployment appsv1.Deployment
		Eventually(func() bool {
			return k8sClient.Get(ctx, nameSpacedDependencyName(runner), &deployment) == nil
		}, timeout, interval).Should(BeTrue())

		//
		Expect(deployment.OwnerReferences).NotTo(BeEmpty())
		Expect(deployment.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
		Expect(deployment.Annotations).To(HaveKey(configVersionAnnotationKey))
		Expect(deployment.Annotations[configVersionAnnotationKey]).To(BeEquivalentTo(runner.Status.ConfigMapVersion))
	}
}

func caseEnvironmentIsSpecified(tc *testCase) {
	ctx := context.Background()
	tc.Runner.Spec.Environment = []string{"foo=bar"}
	tc.CheckRunner = func(runner *v1beta2.Runner) {
		var deployment appsv1.Deployment
		Eventually(func() bool {
			return k8sClient.Get(ctx, nameSpacedDependencyName(runner), &deployment) == nil
		}, timeout, interval).Should(BeTrue())
	}
}

// caseTagsChanged changes the tag list of a managed runner; the operator must
// recreate it and obtain a fresh token.
func caseTagsChanged(tc *testCase) {
	ctx := context.Background()
	tc.CheckRunner = func(runner *v1beta2.Runner) {
		oldAuth := runner.Status.AuthenticationToken

		// update tags
		runner.Spec.Authentication.CreateOptions.TagList = []string{"new", "tag", "list"}
		Expect(k8sClient.Update(ctx, runner)).To(Succeed())

		// runner should get a new authentication token
		newRunner := &v1beta2.Runner{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nameSpacedRunnerName(runner), newRunner)
			return err == nil && newRunner.Status.AuthenticationToken != oldAuth
		}, timeout, interval).Should(BeTrue())
	}
}

// caseRegistrationTokenChanged changes a create option (description) and checks
// the operator recreates the managed runner (the registration hash changes).
func caseRegistrationTokenChanged(tc *testCase) {
	ctx := context.Background()
	tc.CheckRunner = func(runner *v1beta2.Runner) {
		oldHash := runner.Status.RegistrationHash

		runner.Spec.Authentication.CreateOptions.Description = "changed description"
		Expect(k8sClient.Update(ctx, runner)).To(Succeed())

		newRunner := &v1beta2.Runner{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nameSpacedRunnerName(runner), newRunner)
			return err == nil && newRunner.Status.RegistrationHash != oldHash
		}, timeout, interval).Should(BeTrue())
	}
}

func caseSpecChanged(tc *testCase) {
	ctx := context.Background()
	tc.CheckRunner = func(runner *v1beta2.Runner) {
		oldConfigMapVersion := runner.Status.ConfigMapVersion
		dp := getChangedDeployment(ctx, nameSpacedDependencyName(runner), "")
		configMap := getChangedConfigMap(ctx, nameSpacedDependencyName(runner), "")

		// update runner spec
		runner.Spec.Concurrent = 2
		Expect(k8sClient.Update(ctx, runner)).To(Succeed())

		var newRunner v1beta2.Runner
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nameSpacedRunnerName(runner), &newRunner)
			return err == nil && newRunner.Status.Ready && newRunner.Status.ConfigMapVersion != oldConfigMapVersion
		}, timeout, interval).Should(BeTrue())

		// wait until the configmap is updated and fetch the new version
		Eventually(func() bool {
			var cm corev1.ConfigMap
			if err := k8sClient.Get(ctx, nameSpacedDependencyName(&newRunner), &cm); err != nil {
				return false
			}
			return configMap.Data[configMapKeyName] != cm.Data[configMapKeyName]
		}, timeout, interval).Should(BeTrue(), "configmap should have new toml config")

		// verify that our deployment has been amended with a new version
		Eventually(func() bool {
			var newDp appsv1.Deployment
			if err := k8sClient.Get(ctx, nameSpacedDependencyName(&newRunner), &newDp); err != nil {
				return false
			}
			return dp.Annotations[configVersionAnnotationKey] == runner.Status.ConfigMapVersion
		}, timeout, interval).Should(BeTrue(), "deployment should have a new config map version")
	}
}

// /////////////////////////////////////////////
// HELPER FUNCS
// /////////////////////////////////////////////
func getChangedDeployment(ctx context.Context, name types.NamespacedName, resourceDiffersFrom string) *appsv1.Deployment {
	var dp appsv1.Deployment
	Eventually(func() bool {
		if err := k8sClient.Get(ctx, name, &dp); err != nil {
			return false
		}
		return dp.ObjectMeta.ResourceVersion != resourceDiffersFrom
	}, timeout, interval).Should(BeTrue())
	return &dp
}

func getChangedConfigMap(ctx context.Context, name types.NamespacedName, resourceDiffersFrom string) *corev1.ConfigMap {
	var configMap corev1.ConfigMap
	Eventually(func() bool {
		if err := k8sClient.Get(ctx, name, &configMap); err != nil {
			return false
		}
		return configMap.ObjectMeta.ResourceVersion != resourceDiffersFrom
	}, timeout, interval).Should(BeTrue())
	return &configMap
}

func nameSpacedRunnerName(runner *v1beta2.Runner) types.NamespacedName {
	return types.NamespacedName{Name: runner.Name, Namespace: runner.GetNamespace()}
}
func nameSpacedDependencyName(runner *v1beta2.Runner) types.NamespacedName {
	return types.NamespacedName{Name: runner.ChildName(), Namespace: runner.GetNamespace()}
}
func CreateNamespace(c client.Client) (string, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ctrl-test-",
		},
	}
	var err error
	err = wait.Poll(time.Second, 10*time.Second, func() (bool, error) {
		err = c.Create(context.Background(), ns)
		return err == nil, nil
	})
	if err != nil {
		return "", err
	}
	return ns.Name, nil
}

// defaultRunner returns a managed runner (operator creates it on GitLab via the
// access-token path), exercising the mocked CreateRunner call.
func defaultRunner(name string, nameSpace string) *v1beta2.Runner {
	return &v1beta2.Runner{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta2.GroupVersion.String(),
			Kind:       "Runner",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nameSpace,
		},
		Spec: v1beta2.RunnerSpec{
			Authentication: v1beta2.GitlabAuth{
				AccessToken: "glpat-test-access-token",
				CreateOptions: &v1beta2.RunnerCreateOptions{
					RunnerType: "instance_type",
					TagList:    []string{"default-tag"},
				},
			},
		},
	}
}
