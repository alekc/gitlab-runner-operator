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

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"gitlab.k8s.alekc.dev/api/v1beta1"
	"gitlab.k8s.alekc.dev/internal/generate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:docs-gen:collapse=Imports

type testCase struct {
	Runner         *v1beta1.Runner
	CheckCondition func(*v1beta1.Runner) bool
	CheckRunner    func(*v1beta1.Runner)
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
		Expect(k8sClient.Delete(context.Background(), &v1beta1.Runner{
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

	table.DescribeTable(
		"When reconciling Gitlab Runner",
		func(tweaks ...testCaseTweak) {
			tc := &testCase{
				Runner: defaultRunner(RunnerName, RunnerNamespace),
				CheckCondition: func(runner *v1beta1.Runner) bool {
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
			createdRunner := &v1beta1.Runner{}
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
		table.Entry("Should have created a different registration on tag update", caseTagsChanged),
		table.Entry("Should have updated runner status with auth token", caseTestAuthToken),
	)
})

func caseTestAuthToken(tc *testCase) {
	tc.CheckRunner = func(runner *v1beta1.Runner) {
		Expect(runner.Status.Error).To(BeEmpty())
		Expect(runner.Status.AuthenticationToken).To(BeEquivalentTo("95ef6f888cb2280a3a070186cf55b04f"))
	}
}

func caseGeneratedConfigMap(tc *testCase) {
	var configMap corev1.ConfigMap
	ctx := context.Background()

	tc.CheckCondition = func(runner *v1beta1.Runner) bool {
		return runner.Status.ConfigMapVersion != ""
	}

	tc.CheckRunner = func(runner *v1beta1.Runner) {
		Eventually(func() bool {
			return k8sClient.Get(ctx, nameSpacedDependencyName(runner), &configMap) == nil
		}, timeout, interval).Should(BeTrue())

		Expect(configMap.OwnerReferences).NotTo(BeEmpty())
		Expect(configMap.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
		Expect(configMap.Data).Should(HaveKey(configMapKeyName), "Child config map should have %s data entry", configMapKeyName)
		Expect(runner.Status.ConfigMapVersion).Should(
			BeEquivalentTo(configMap.Annotations[configVersionAnnotationKey]),
			"runner.Status.ConfigMapVersion should report the same version as configmap",
		)
	}
}

func caseCheckDeployment(tc *testCase) {
	ctx := context.Background()
	tc.CheckRunner = func(runner *v1beta1.Runner) {
		var deployment appsv1.Deployment
		Eventually(func() bool {
			return k8sClient.Get(ctx, nameSpacedDependencyName(runner), &deployment) == nil
		}, timeout, interval).Should(BeTrue())

		Expect(deployment.OwnerReferences).NotTo(BeEmpty())
		Expect(deployment.OwnerReferences[0].UID).To(BeEquivalentTo(runner.UID))
		Expect(deployment.Annotations).To(HaveKey(configVersionAnnotationKey))
		Expect(deployment.Annotations[configVersionAnnotationKey]).To(BeEquivalentTo(runner.Status.ConfigMapVersion))

		// todo: verify number of running pods
	}
}

// caseTagsChanged deals with situation when we change tags for an existing runner
func caseTagsChanged(tc *testCase) {
	ctx := context.Background()
	tc.CheckRunner = func(runner *v1beta1.Runner) {
		oldAuth := runner.Status.AuthenticationToken

		// update tags
		runner.Spec.RegistrationConfig.TagList = []string{"new", "tag", "list"}
		Expect(k8sClient.Update(ctx, runner)).To(Succeed())

		// runner should get a new hash version
		newRunner := &v1beta1.Runner{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nameSpacedRunnerName(runner), newRunner)
			return err == nil && newRunner.Status.AuthenticationToken != oldAuth
		}, timeout, interval).Should(BeTrue())
	}
}

func caseSpecChanged(tc *testCase) {
	ctx := context.Background()
	tc.CheckRunner = func(runner *v1beta1.Runner) {
		oldConfigMapVersion := runner.Status.ConfigMapVersion
		dp := getChangedDeployment(ctx, nameSpacedDependencyName(runner), "")
		configMap := getChangedConfigMap(ctx, nameSpacedDependencyName(runner), "")

		// update runner spec
		runner.Spec.Concurrent = 2
		Expect(k8sClient.Update(ctx, runner)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, nameSpacedRunnerName(runner),
				runner)
			return err == nil && runner.Status.ConfigMapVersion != oldConfigMapVersion
		}, timeout, interval).Should(BeTrue())

		// wait until the configmap is updated and fetch the new version
		configMap = getChangedConfigMap(ctx, nameSpacedDependencyName(runner), configMap.ResourceVersion)
		Expect(configMap.Annotations[configVersionAnnotationKey]).To(BeEquivalentTo(runner.Status.ConfigMapVersion))

		// verify that our deployment has been amended with a new version
		dp = getChangedDeployment(ctx, nameSpacedDependencyName(runner), dp.ResourceVersion)
		Expect(dp.Annotations[configVersionAnnotationKey]).To(BeEquivalentTo(runner.Status.ConfigMapVersion))
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

func nameSpacedRunnerName(runner *v1beta1.Runner) types.NamespacedName {
	return types.NamespacedName{Name: runner.Name, Namespace: runner.Namespace}
}
func nameSpacedDependencyName(runner *v1beta1.Runner) types.NamespacedName {
	return types.NamespacedName{Name: runner.ChildName(), Namespace: runner.Namespace}
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

// defaultRunner returns an instance of gitlab runner with default values
func defaultRunner(name string, nameSpace string) *v1beta1.Runner {
	return &v1beta1.Runner{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.GroupVersion.String(),
			Kind:       "Runner",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nameSpace,
		},
		Spec: v1beta1.RunnerSpec{
			RegistrationConfig: v1beta1.RegisterNewRunnerOptions{
				Token:   pointer.StringPtr("zTS6g2Q8bp8y13_ynfpN"),
				TagList: []string{"default-tag"},
			},
		},
	}
}
