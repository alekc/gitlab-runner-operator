package v1beta1

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var _ = Describe("Runner controller", func() {

	const (
		RunnerName      = "test-runner"
		RunnerNamespace = "default"

		duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	// namespacedDependencyName := types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}

	Context("Initially", func() {
		var runner Runner
		It("It should be created with no errors", func() {
			runner = Runner{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       "Runner",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      RunnerName,
					Namespace: RunnerNamespace,
				},
				Spec: RunnerSpec{
					RegistrationConfig: RegisterNewRunnerOptions{
						Token:   pointer.StringPtr("rYwg6EogqxSuvsFCVvAT"),
						TagList: []string{"testing-runner-operator"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &runner)).Should(Succeed())
		})
		It("Should have expected default values", func() {
			Expect(runner.Spec.GitlabInstanceURL).To(Equal("https://gitlab.com/"))
		})
	})

	// Context("When runner already exists", func() {
	// 	namespacedDependencyName := types.NamespacedName{Name: RunnerName, Namespace: RunnerNamespace}
	//
	// 	It("It should not permit to change the runner's name", func() {
	// 		var runner Runner
	// 		ctx := context.TODO()
	//
	// 		// obtain the existing client
	// 		err := k8sClient.Get(ctx, namespacedDependencyName, &runner)
	// 		Expect(err).To(BeNil())
	//
	// 		// // runner.ObjectMeta.Name = fmt.Sprintf("%s-new-name", runner.ObjectMeta.Name)
	// 		// runner.Spec.Concurrent = 2
	// 		// err = k8sClient.Update(ctx, &runner)
	// 		// Expect(err).ToNot(BeNil())
	//
	// 		// newRunner := &Runner{
	// 		// 	TypeMeta: metav1.TypeMeta{
	// 		// 		APIVersion: GroupVersion.String(),
	// 		// 		Kind:       "Runner",
	// 		// 	},
	// 		// 	ObjectMeta: metav1.ObjectMeta{
	// 		// 		Name:      RunnerName,
	// 		// 		Namespace: RunnerNamespace,
	// 		// 	},
	// 		// 	Spec: RunnerSpec{
	// 		// 		RegistrationConfig: RegisterNewRunnerOptions{
	// 		// 			Token:   pointer.StringPtr("rYwg6EogqxSuvsFCVvAT"),
	// 		// 			TagList: []string{"testing-runner-operator"},
	// 		// 		},
	// 		// 	},
	// 		// }
	// 		// Expect(k8sClient.Create(ctx, newRunner)).Should(Succeed())
	// 	})
	// })
})
