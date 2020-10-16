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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gitlabRunOp "go.alekc.dev/gitlab-runner-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
The first step to writing a simple integration test is to actually create an instance of CronJob you can run tests against.
Note that to create a CronJob, you’ll need to create a stub CronJob struct that contains your CronJob’s specifications.

Note that when we create a stub CronJob, the CronJob also needs stubs of its required downstream objects.
Without the stubbed Job template spec and the Pod template spec below, the Kubernetes API will not be able to
create the CronJob.
*/
var _ = Describe("CronJob controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		RunnerName      = "test-runner"
		RunnerNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a runner type", func() {
		It("should spin a gitlab runner pod", func() {
			By("By creating a new CronJob")
			ctx := context.Background()
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

			lookupKey := types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}
			runner := &gitlabRunOp.Runner{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				return k8sClient.Get(ctx, lookupKey, runner) == nil
			}, timeout, interval).Should(BeTrue())

		})
	})
})
