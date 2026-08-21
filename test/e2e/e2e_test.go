package e2e

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const sharedExecutorClusterRole = "gitlab-runner-operator-executor"

// optionalPDBClusterRole carries the pod_disruption_budget grant on its own.
const optionalPDBClusterRole = "gitlab-runner-operator-executor-pdb"

// e2eVerbsFor returns the verbs granted on resource, or nil when ungranted.
func e2eVerbsFor(rules []rbacv1.PolicyRule, resource string) []string {
	for _, r := range rules {
		for _, res := range r.Resources {
			if res == resource {
				return r.Verbs
			}
		}
	}
	return nil
}

func objMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: e2eNamespace}
}

func key(name string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: e2eNamespace}
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func managedCreateOptions(tags []string) *gitlabv1beta2.RunnerCreateOptions {
	return &gitlabv1beta2.RunnerCreateOptions{
		RunnerType:  "project_type",
		ProjectID:   ptr.To(projectID),
		RunUntagged: ptr.To(false),
		TagList:     tags,
	}
}

func waitRunnerReady(name string) gitlabv1beta2.Runner {
	GinkgoHelper()
	var got gitlabv1beta2.Runner
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(context.Background(), key(name), &got)).To(Succeed())
		g.Expect(got.Status.Ready).To(BeTrue(), "runner not Ready (status error: %q)", got.Status.Error)
	}, timeout, interval).Should(Succeed())
	return got
}

func waitDeploymentUp(childName string) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		var dep appsv1.Deployment
		g.Expect(k8sClient.Get(context.Background(), key(childName), &dep)).To(Succeed())
		g.Expect(dep.Status.AvailableReplicas).To(BeNumerically(">=", 1))
	}, timeout, interval).Should(Succeed())
}

func waitGone(name string) {
	GinkgoHelper()
	Eventually(func() bool {
		return apierrors.IsNotFound(k8sClient.Get(context.Background(), key(name), &gitlabv1beta2.Runner{}))
	}, timeout, interval).Should(BeTrue())
}

func triggerPipeline() int64 {
	GinkgoHelper()
	// RUNNER_TAG pins build-job to this run's runner. Without it the job takes
	// the CI file's default tag, which every concurrent run shares.
	p, _, err := glab.Pipelines.CreatePipeline(projectID, &gitlab.CreatePipelineOptions{
		Ref: gitlab.Ptr(defaultBranch),
		Variables: &[]*gitlab.PipelineVariableOptions{{
			Key:   gitlab.Ptr("RUNNER_TAG"),
			Value: gitlab.Ptr(jobTag),
		}},
	})
	Expect(err).NotTo(HaveOccurred(), "could not trigger a pipeline (token needs api scope + Developer role)")
	return p.ID
}

// waitJobRanOnRunner waits until build-job in the pipeline reaches success on
// the given runner id, proving the operator-managed runner executed real CI.
func waitJobRanOnRunner(pipelineID int64, runnerID int) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		jobs, _, err := glab.Jobs.ListPipelineJobs(projectID, pipelineID, nil)
		g.Expect(err).NotTo(HaveOccurred())
		var build *gitlab.Job
		for _, j := range jobs {
			if j.Name == "build-job" {
				build = j
			}
		}
		g.Expect(build).NotTo(BeNil(), "build-job not present yet")
		g.Expect(build.Status).NotTo(BeElementOf("failed", "canceled"), "build-job ended badly")
		g.Expect(build.Status).To(Equal("success"), "build-job not finished yet")
		g.Expect(build.Runner.ID).To(Equal(int64(runnerID)), "build-job must run on our runner")
	}, timeout, interval).Should(Succeed())
}

var _ = Describe("Managed runner executes a real CI job", func() {
	It("registers on GitLab, becomes Ready, and runs build-job to success on our runner", func() {
		ctx := context.Background()
		name := uniqueName("e2e-managed")
		runner := &gitlabv1beta2.Runner{
			ObjectMeta: objMeta(name),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication: gitlabv1beta2.GitlabAuth{
					AccessToken:   &gitlabv1beta2.TokenSource{Value: gitlabToken},
					CreateOptions: managedCreateOptions([]string{jobTag}),
				},
			},
		}
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
		DeferCleanup(func() { deleteRunnerCR(name) })

		By("becoming Ready with a GitLab-assigned runner id")
		got := waitRunnerReady(name)
		Expect(got.Status.RunnerID).NotTo(BeZero())

		By("bringing up the runner manager Deployment")
		waitDeploymentUp(got.ChildName())

		By("triggering a pipeline and asserting build-job succeeds on our runner")
		waitJobRanOnRunner(triggerPipeline(), got.Status.RunnerID)

		By("deleting the Runner and confirming the finalizer completes")
		Expect(k8sClient.Delete(ctx, &got)).To(Succeed())
		waitGone(name)
	})
})

var _ = Describe("Bring-your-own-token runner from a Secret", func() {
	It("becomes Ready from a minted glrt- token read via secret_key_ref, without the operator creating a runner", func() {
		ctx := context.Background()
		runnerID, token := mintRunner([]string{jobTag})
		DeferCleanup(func() { deleteRunnerByID(runnerID) })

		name := uniqueName("e2e-byo")
		secretName := name + "-token"
		Expect(k8sClient.Create(ctx, &corev1.Secret{
			ObjectMeta: objMeta(secretName),
			StringData: map[string]string{"token": token},
		})).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, &corev1.Secret{ObjectMeta: objMeta(secretName)}) })

		runner := &gitlabv1beta2.Runner{
			ObjectMeta: objMeta(name),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication: gitlabv1beta2.GitlabAuth{
					Token: &gitlabv1beta2.TokenSource{
						SecretKeyRef: &gitlabv1beta2.SecretKeySelector{Name: secretName},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
		DeferCleanup(func() { deleteRunnerCR(name) })

		got := waitRunnerReady(name)
		Expect(got.Status.RunnerID).To(BeZero(), "BYO runners are not created by the operator")
		waitDeploymentUp(got.ChildName())

		By("rendering the token into the config Secret (config.toml present)")
		var cfg corev1.Secret
		Expect(k8sClient.Get(ctx, key(got.ChildName()), &cfg)).To(Succeed())
		Expect(cfg.Data).To(HaveKey("config.toml"))

		By("triggering a pipeline and asserting build-job runs on the BYO runner")
		waitJobRanOnRunner(triggerPipeline(), runnerID)
	})
})

var _ = Describe("MultiRunner", func() {
	It("registers every managed entry and shares a single ServiceAccount", func() {
		ctx := context.Background()
		name := uniqueName("e2e-multi")
		mr := &gitlabv1beta2.MultiRunner{
			ObjectMeta: objMeta(name),
			Spec: gitlabv1beta2.MultiRunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Entries: []gitlabv1beta2.MultiRunnerEntry{
					{Name: "one", Authentication: gitlabv1beta2.GitlabAuth{
						AccessToken:   &gitlabv1beta2.TokenSource{Value: gitlabToken},
						CreateOptions: managedCreateOptions([]string{jobTag}),
					}},
					{Name: "two", Authentication: gitlabv1beta2.GitlabAuth{
						AccessToken:   &gitlabv1beta2.TokenSource{Value: gitlabToken},
						CreateOptions: managedCreateOptions([]string{"e2e-extra"}),
					}},
				},
			},
		}
		Expect(k8sClient.Create(ctx, mr)).To(Succeed())
		DeferCleanup(func() {
			_ = k8sClient.Delete(ctx, &gitlabv1beta2.MultiRunner{ObjectMeta: objMeta(name)})
		})

		By("both entries registering and the object becoming Ready")
		var got gitlabv1beta2.MultiRunner
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, key(name), &got)).To(Succeed())
			g.Expect(got.Status.Ready).To(BeTrue(), "multirunner not Ready (status error: %q)", got.Status.Error)
			g.Expect(got.Status.RunnerIDs["one"]).NotTo(BeZero())
			g.Expect(got.Status.RunnerIDs["two"]).NotTo(BeZero())
		}, timeout, interval).Should(Succeed())
		Expect(got.Status.RunnerIDs["one"]).NotTo(Equal(got.Status.RunnerIDs["two"]),
			"each managed entry must get its own GitLab runner id")

		waitDeploymentUp(got.ChildName())

		By("provisioning a single shared ServiceAccount")
		var sa corev1.ServiceAccount
		Expect(k8sClient.Get(ctx, key(got.ChildName()), &sa)).To(Succeed())

		By("running the shared Deployment under that ServiceAccount")
		var dep appsv1.Deployment
		Expect(k8sClient.Get(ctx, key(got.ChildName()), &dep)).To(Succeed())
		Expect(dep.Spec.Template.Spec.ServiceAccountName).To(Equal(got.ChildName()))

		By("removing both runners from GitLab on delete")
		idOne, idTwo := got.Status.RunnerIDs["one"], got.Status.RunnerIDs["two"]
		Expect(k8sClient.Delete(ctx, &got)).To(Succeed())
		Eventually(func() bool {
			return apierrors.IsNotFound(k8sClient.Get(ctx, key(name), &gitlabv1beta2.MultiRunner{}))
		}, timeout, interval).Should(BeTrue())

		By("confirming both runners are gone from GitLab (404)")
		for _, id := range []int{idOne, idTwo} {
			Eventually(func(g Gomega) {
				_, resp, err := glab.Runners.GetRunnerDetails(id)
				g.Expect(err).To(HaveOccurred())
				g.Expect(resp).NotTo(BeNil())
				g.Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			}, timeout, interval).Should(Succeed())
		}
	})
})

var _ = Describe("Live spec change", func() {
	It("recreates the managed runner on GitLab when create_options change", func() {
		ctx := context.Background()
		name := uniqueName("e2e-change")
		runner := &gitlabv1beta2.Runner{
			ObjectMeta: objMeta(name),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication: gitlabv1beta2.GitlabAuth{
					AccessToken:   &gitlabv1beta2.TokenSource{Value: gitlabToken},
					CreateOptions: managedCreateOptions([]string{jobTag}),
				},
			},
		}
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
		DeferCleanup(func() { deleteRunnerCR(name) })

		oldID := waitRunnerReady(name).Status.RunnerID
		Expect(oldID).NotTo(BeZero())

		By("editing create_options (tags), which changes the registration hash")
		Eventually(func(g Gomega) {
			var cur gitlabv1beta2.Runner
			g.Expect(k8sClient.Get(ctx, key(name), &cur)).To(Succeed())
			cur.Spec.Authentication.CreateOptions.TagList = []string{jobTag, "changed"}
			g.Expect(k8sClient.Update(ctx, &cur)).To(Succeed())
		}, timeout, interval).Should(Succeed())

		By("the operator recreating the runner (a new GitLab id) and staying Ready")
		Eventually(func(g Gomega) {
			var cur gitlabv1beta2.Runner
			g.Expect(k8sClient.Get(ctx, key(name), &cur)).To(Succeed())
			g.Expect(cur.Status.Ready).To(BeTrue())
			g.Expect(cur.Status.RunnerID).NotTo(BeZero())
			g.Expect(cur.Status.RunnerID).NotTo(Equal(oldID), "expected a new runner id after the change")
		}, timeout, interval).Should(Succeed())
	})
})

var _ = Describe("RBAC provisioning", func() {
	It("creates the per-runner ServiceAccount and a RoleBinding to the shared executor ClusterRole", func() {
		ctx := context.Background()
		name := uniqueName("e2e-rbac")
		runner := &gitlabv1beta2.Runner{
			ObjectMeta: objMeta(name),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication: gitlabv1beta2.GitlabAuth{
					AccessToken:   &gitlabv1beta2.TokenSource{Value: gitlabToken},
					CreateOptions: managedCreateOptions([]string{jobTag}),
				},
			},
		}
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
		DeferCleanup(func() { deleteRunnerCR(name) })
		got := waitRunnerReady(name)
		child := got.ChildName()

		By("the shared executor ClusterRole existing with the rules the operator computed")
		var executorRole rbacv1.ClusterRole
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: sharedExecutorClusterRole}, &executorRole)).To(Succeed())
		// CI builds a fresh cluster, so this pins what the operator creates,
		// not an upgrade. Drift convergence is covered in internal/crud.
		Expect(e2eVerbsFor(executorRole.Rules, "secrets")).To(ContainElement("create"),
			"executor ClusterRole is missing the base rules")
		// The optional grant must not ride the shared role.
		Expect(e2eVerbsFor(executorRole.Rules, "poddisruptionbudgets")).To(BeEmpty(),
			"the shared executor ClusterRole still grants poddisruptionbudgets unconditionally")

		By("the optional pod_disruption_budget ClusterRole existing, bound to nobody")
		var pdbRole rbacv1.ClusterRole
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: optionalPDBClusterRole}, &pdbRole)).To(Succeed())
		Expect(e2eVerbsFor(pdbRole.Rules, "poddisruptionbudgets")).To(ConsistOf("get", "create"),
			"optional ClusterRole is missing the poddisruptionbudgets rule")
		Consistently(func() bool {
			return apierrors.IsNotFound(k8sClient.Get(ctx,
				types.NamespacedName{Namespace: runner.GetNamespace(), Name: "pdb-" + child},
				&rbacv1.RoleBinding{}))
		}, "5s", interval).Should(BeTrue(),
			"an optional binding was created for a runner that never set pod_disruption_budget")

		By("a per-runner ServiceAccount existing")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, key(child), &corev1.ServiceAccount{})).To(Succeed())
		}, timeout, interval).Should(Succeed())

		By("a RoleBinding binding that ServiceAccount to the shared ClusterRole")
		Eventually(func(g Gomega) {
			var rbs rbacv1.RoleBindingList
			g.Expect(k8sClient.List(ctx, &rbs, client.InNamespace(e2eNamespace))).To(Succeed())
			found := false
			for _, rb := range rbs.Items {
				if rb.RoleRef.Kind != "ClusterRole" || rb.RoleRef.Name != sharedExecutorClusterRole {
					continue
				}
				for _, s := range rb.Subjects {
					if s.Kind == "ServiceAccount" && s.Name == child && s.Namespace == e2eNamespace {
						found = true
					}
				}
			}
			g.Expect(found).To(BeTrue(), "no RoleBinding binds SA %q to ClusterRole %q", child, sharedExecutorClusterRole)
		}, timeout, interval).Should(Succeed())
	})
})
