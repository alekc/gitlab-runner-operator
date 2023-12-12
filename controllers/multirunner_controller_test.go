package controllers

//
// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"strconv"
// 	"time"
//
// 	"gitlab.k8s.alekc.dev/api/v1beta1"
// 	"gitlab.k8s.alekc.dev/internal/generate"
// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/types"
// 	"k8s.io/utils/pointer"
// 	"sigs.k8s.io/controller-runtime/pkg/client"
//
// 	. "github.com/onsi/ginkgo"
// 	"github.com/onsi/ginkgo/extensions/table"
// 	. "github.com/onsi/gomega"
// )
//
// // +kubebuilder:docs-gen:collapse=Imports
//
// type multiRunnerTestCase struct {
// 	MutliRunner    *v1beta1.MultiRunner
// 	CheckCondition func(*v1beta1.MultiRunner) bool
// 	CheckRunner    func(*v1beta1.MultiRunner)
// }
//
// type multiRunnerTestCaseTweak func(runnerTestCase *multiRunnerTestCase)
//
// var _ = Describe("Runner controller", func() {
//
// 	// Define utility constants for object names and testing timeouts/durations and intervals.
// 	var RunnerNamespace string
// 	var RunnerName string
//
// 	// if we are running a debugger, then increase timeout to 10 minutes to prevent killed debug sessions
// 	if val, ok := os.LookupEnv("TEST_TIMEOUT"); ok {
// 		newTimeout, err := strconv.Atoi(val)
// 		if err == nil {
// 			timeout = time.Second * time.Duration(newTimeout)
// 		}
// 	}
//
// 	// before every test
// 	BeforeEach(func() {
// 		var err error
// 		// create a runner namespace
// 		RunnerNamespace, err = CreateNamespace(k8sClient)
// 		Expect(err).ToNot(HaveOccurred())
//
// 		// generate a random name for the runner
// 		RunnerName = fmt.Sprintf("test-runner-%s", generate.RandomString(5))
// 	})
//
// 	// after each test perform some cleaning actions
// 	AfterEach(func() {
// 		// delete the runner
// 		Expect(k8sClient.Delete(context.Background(), &v1beta1.MultiRunner{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      RunnerName,
// 				Namespace: RunnerNamespace,
// 			},
// 		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
//
// 		// delete namespace
// 		Expect(k8sClient.Delete(context.Background(), &corev1.Namespace{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: RunnerNamespace,
// 			},
// 		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
// 	})
//
// 	table.DescribeTable(
// 		"When reconciling Gitlab Runner",
// 		func(tweaks ...multiRunnerTestCaseTweak) {
// 			tc := &multiRunnerTestCase{
// 				MutliRunner: defaultMultiRunner(RunnerName, RunnerNamespace),
// 				CheckCondition: func(runner *v1beta1.MultiRunner) bool {
// 					return runner.UID != "" && runner.Status.Ready
// 				},
// 			}
// 			for _, tweak := range tweaks {
// 				tweak(tc)
// 			}
//
// 			ctx := context.Background()
// 			By("creating a runner")
// 			Expect(Expect(k8sClient.Create(ctx, tc.MutliRunner)).To(Succeed()))
//
// 			//
// 			esKey := types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}
// 			createdRunner := &v1beta1.MultiRunner{}
// 			By("checking the runner condition")
// 			Eventually(func() bool {
// 				err := k8sClient.Get(ctx, esKey, createdRunner)
// 				if err != nil {
// 					return false
// 				}
// 				return tc.CheckCondition(createdRunner)
// 			}, timeout, interval).Should(BeTrue())
//
// 			//
// 			tc.CheckRunner(createdRunner)
// 		},
// 		// table.Entry("Should register", caseEnvironmentIsSpecified),
// 		// table.Entry("Should have created a different registration on tag update", caseTagsChanged),
// 		// table.Entry("Should have created a different registration on registration token update", caseRegistrationTokenChanged),
// 		// table.Entry("Should have updated runner status with auth token", caseTestAuthToken),
// 		// table.Entry("Should have created required RBAC", caseRBACCheck),
// 		// table.Entry("Should have generated config map", caseGeneratedConfigMap),
// 		// table.Entry("Should have generated deployment", caseCheckDeployment),
// 		// table.Entry("On spec change, config map should be updated", caseSpecChanged),
// 	)
// })
//
// func defaultMultiRunner(name string, nameSpace string) *v1beta1.MultiRunner {
// 	return &v1beta1.MultiRunner{
// 		TypeMeta: metav1.TypeMeta{
// 			APIVersion: v1beta1.GroupVersion.String(),
// 			Kind:       "MultiRunner",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: nameSpace,
// 		},
// 		Spec: v1beta1.MultiRunnerSpec{
// 			Entries: []v1beta1.MultiRunnerEntry{
// 				{
// 					RegistrationConfig: v1beta1.RegisterNewRunnerOptions{
// 						Token:   pointer.String("zTS6g2Q8bp8y13_ynfpN"),
// 						TagList: []string{"default-tag"},
// 					},
// 				},
// 			},
// 		},
// 	}
// }
