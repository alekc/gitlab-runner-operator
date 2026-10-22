package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// drainJobSeconds leaves well over 30s of job after the new manager turns Ready
// and the old one is stopped, so surviving the stop proves a real drain.
const drainJobSeconds = 120

// drainCIFile runs one job long enough to be caught mid-flight by a stop.
const drainCIFile = `drain-job:
  image: alpine:3.22
  tags:
    - $RUNNER_TAG
  script:
    - echo "drain-job on runner $CI_RUNNER_ID"
    - sleep %d
    - echo "drain-job finished"
`

// drainBranch commits a CI file with a long job to a throwaway branch. The
// fixture's .gitlab-ci.yml is OpenTofu-managed, so the suite does not edit it.
func drainBranch() string {
	GinkgoHelper()
	branch := uniqueName("e2e-drain")
	_, _, err := glab.Branches.CreateBranch(projectID, &gitlab.CreateBranchOptions{
		Branch: gitlab.Ptr(branch), Ref: gitlab.Ptr(defaultBranch),
	})
	Expect(err).NotTo(HaveOccurred(), "could not create branch %s", branch)
	DeferCleanup(func() { _, _ = glab.Branches.DeleteBranch(projectID, branch) })

	_, _, err = glab.RepositoryFiles.UpdateFile(projectID, ".gitlab-ci.yml", &gitlab.UpdateFileOptions{
		Branch:        gitlab.Ptr(branch),
		Content:       gitlab.Ptr(fmt.Sprintf(drainCIFile, drainJobSeconds)),
		CommitMessage: gitlab.Ptr("e2e: long-running drain-job"),
	})
	Expect(err).NotTo(HaveOccurred(), "could not commit the drain CI file")
	return branch
}

// drainTag is this spec's own tag: a manager left by an earlier spec still
// advertises jobTag while it shuts down, and must not take the drain job.
var drainTag = jobTag + "-drain"

func triggerDrainPipeline(branch string) int64 {
	GinkgoHelper()
	p, _, err := glab.Pipelines.CreatePipeline(projectID, &gitlab.CreatePipelineOptions{
		Ref: gitlab.Ptr(branch),
		Variables: &[]*gitlab.PipelineVariableOptions{{
			Key: gitlab.Ptr("RUNNER_TAG"), Value: gitlab.Ptr(drainTag),
		}},
	})
	Expect(err).NotTo(HaveOccurred(), "could not trigger the drain pipeline")
	return p.ID
}

func drainJob(pipelineID int64) (*gitlab.Job, error) {
	jobs, _, err := glab.Jobs.ListPipelineJobs(projectID, pipelineID, &gitlab.ListJobsOptions{})
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.Name == "drain-job" {
			return j, nil
		}
	}
	return nil, fmt.Errorf("drain-job not present in pipeline %d yet", pipelineID)
}

// waitDrainJob waits for drain-job to reach want, failing fast on a terminal
// status that is not want.
func waitDrainJob(pipelineID int64, want string, budget time.Duration) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		job, err := drainJob(pipelineID)
		g.Expect(err).NotTo(HaveOccurred())
		if job == nil {
			return
		}
		if want != job.Status && (job.Status == "failed" || job.Status == "canceled") {
			StopTrying(fmt.Sprintf("drain-job ended %q while waiting for %q", job.Status, want)).Now()
		}
		g.Expect(job.Status).To(Equal(want))
	}, budget, interval).Should(Succeed())
}

// expectDrainJobOn fails unless drain-job is running on runnerID, so stopping
// this spec's manager is known to interrupt the job under test.
func expectDrainJobOn(pipelineID int64, runnerID int) {
	GinkgoHelper()
	job, err := drainJob(pipelineID)
	Expect(err).NotTo(HaveOccurred())
	Expect(job.Runner.ID).To(Equal(int64(runnerID)), "drain-job is running on another runner")
}

// managerPodName returns the one running, not terminating, manager pod.
func managerPodName(name string) string {
	GinkgoHelper()
	var pods corev1.PodList
	Expect(k8sClient.List(context.Background(), &pods, client.InNamespace(e2eNamespace),
		client.MatchingLabels{"deployment": name})).To(Succeed())
	for _, p := range pods.Items {
		if p.DeletionTimestamp == nil {
			return p.Name
		}
	}
	Fail("no live manager pod for " + name)
	return ""
}

var _ = Describe("Manager drain", func() {
	It("finishes a running job across a manager roll and a Runner delete", func() {
		ctx := context.Background()
		name := uniqueName("e2e-drain")
		runner := &gitlabv1beta2.Runner{
			ObjectMeta: objMeta(name),
			Spec: gitlabv1beta2.RunnerSpec{
				GitlabInstanceURL: gitlabURL,
				Authentication: gitlabv1beta2.GitlabAuth{
					AccessToken:   &gitlabv1beta2.TokenSource{Value: gitlabToken},
					CreateOptions: managedCreateOptions([]string{drainTag}),
				},
			},
		}
		Expect(k8sClient.Create(ctx, runner)).To(Succeed())
		DeferCleanup(func() { deleteRunnerCR(name) })
		got := waitRunnerReady(name)
		waitDeploymentUp(got.ChildName())
		branch := drainBranch()

		By("rolling the manager while drain-job runs")
		pipeline := triggerDrainPipeline(branch)
		waitDrainJob(pipeline, "running", dispatchTimeout)
		expectDrainJobOn(pipeline, got.Status.RunnerID)
		oldPod := managerPodName(name)
		Eventually(func(g Gomega) {
			var cur gitlabv1beta2.Runner
			g.Expect(k8sClient.Get(ctx, key(name), &cur)).To(Succeed())
			cur.Spec.RunnerEnv = append(cur.Spec.RunnerEnv, corev1.EnvVar{Name: "E2E_ROLL", Value: "1"})
			g.Expect(k8sClient.Update(ctx, &cur)).To(Succeed())
		}, timeout, interval).Should(Succeed())

		By("the old manager draining past 30s instead of being killed")
		var stoppedAt time.Time
		Eventually(func(g Gomega) {
			var p corev1.Pod
			g.Expect(k8sClient.Get(ctx, key(oldPod), &p)).To(Succeed())
			g.Expect(p.DeletionTimestamp).NotTo(BeNil(), "old manager pod not stopped yet")
			// deletionTimestamp is the kill deadline: stop time plus grace.
			grace := time.Duration(*p.Spec.TerminationGracePeriodSeconds) * time.Second
			stoppedAt = p.DeletionTimestamp.Add(-grace)
		}, timeout, interval).Should(Succeed())
		waitDrainJob(pipeline, "success", timeout)
		job, err := drainJob(pipeline)
		Expect(err).NotTo(HaveOccurred())
		Expect(job.FinishedAt).NotTo(BeNil())
		Expect(job.FinishedAt.Sub(stoppedAt)).To(BeNumerically(">", 30*time.Second),
			"drain-job finished within 30s of the stop, so this proves nothing about draining")
		Eventually(func() bool {
			return apierrors.IsNotFound(k8sClient.Get(ctx, key(oldPod), &corev1.Pod{}))
		}, timeout, interval).Should(BeTrue(), "old manager pod outlived its finished job")

		By("deleting the Runner mid-job: it stays registered until the job is done")
		waitDeploymentUp(got.ChildName())
		pipeline = triggerDrainPipeline(branch)
		waitDrainJob(pipeline, "running", dispatchTimeout)
		expectDrainJobOn(pipeline, got.Status.RunnerID)
		Expect(k8sClient.Delete(ctx, &gitlabv1beta2.Runner{ObjectMeta: objMeta(name)})).To(Succeed())
		Eventually(func(g Gomega) {
			job, err := drainJob(pipeline)
			g.Expect(err).NotTo(HaveOccurred())
			if job.Status == "running" {
				var cur gitlabv1beta2.Runner
				if err := k8sClient.Get(ctx, key(name), &cur); err != nil {
					StopTrying("the Runner was removed while its job was still running").Wrap(err).Now()
				}
				if _, _, err := glab.Runners.GetRunnerDetails(got.Status.RunnerID); err != nil {
					StopTrying("the runner was deregistered from GitLab mid-job").Wrap(err).Now()
				}
			}
			g.Expect(job.Status).To(Equal("success"))
		}, timeout, interval).Should(Succeed())
		waitGone(name)
	})
})
