package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// managerPodStates describes the Runner's manager pods, terminating ones too,
// so a failure shows whether the manager outlived its job (#136).
func managerPodStates(name string) string {
	var pods corev1.PodList
	if err := k8sClient.List(context.Background(), &pods, client.InNamespace(e2eNamespace),
		client.MatchingLabels{"deployment": name}); err != nil {
		return "pods: " + err.Error()
	}
	if len(pods.Items) == 0 {
		return "pods: none"
	}
	var b strings.Builder
	for _, p := range pods.Items {
		fmt.Fprintf(&b, "pod %s phase=%s", p.Name, p.Status.Phase)
		for _, o := range p.OwnerReferences {
			fmt.Fprintf(&b, " owner=%s/%s", o.Kind, o.Name)
		}
		if p.DeletionTimestamp != nil {
			fmt.Fprintf(&b, " kill-deadline=%s", p.DeletionTimestamp.UTC().Format(time.TimeOnly))
		}
		for _, cs := range p.Status.ContainerStatuses {
			switch {
			case cs.State.Terminated != nil:
				t := cs.State.Terminated
				fmt.Fprintf(&b, " %s=terminated(%s exit=%d at=%s)", cs.Name, t.Reason, t.ExitCode,
					t.FinishedAt.UTC().Format(time.TimeOnly))
			case cs.State.Running != nil:
				fmt.Fprintf(&b, " %s=running", cs.Name)
			}
		}
		b.WriteString("; ")
	}
	return b.String()
}

// describeJob renders the fields that tell a lost drain-job from a late one.
func describeJob(j *gitlab.Job) string {
	s := fmt.Sprintf("job %d status=%s", j.ID, j.Status)
	if j.StartedAt != nil {
		s += " started=" + j.StartedAt.UTC().Format(time.TimeOnly)
	}
	if j.FinishedAt != nil {
		s += " finished=" + j.FinishedAt.UTC().Format(time.TimeOnly)
	}
	if j.FailureReason != "" {
		s += " failure_reason=" + j.FailureReason
	}
	return s
}

var terminalJobStatus = map[string]bool{
	"success": true, "failed": true, "canceled": true, "skipped": true,
}

// settleDrainJob polls drain-job for up to budget and says where it ended: a
// late success means GitLab lagged, a failure means the job was lost mid-run.
// The manager pods keep being sampled meanwhile, so the timeline does not stop.
func settleDrainJob(pipelineID int64, name string, budget time.Duration) string {
	deadline := time.Now().Add(budget)
	last := ""
	for {
		if pods := managerPodStates(name); pods != last {
			GinkgoWriter.Printf("%s settling | %s\n", time.Now().UTC().Format(time.TimeOnly), pods)
			last = pods
		}
		job, err := drainJob(pipelineID)
		if err == nil && terminalJobStatus[job.Status] {
			return "settled: " + describeJob(job)
		}
		if time.Now().After(deadline) {
			if err != nil {
				return "unsettled: " + err.Error()
			}
			return fmt.Sprintf("unsettled after %s: %s", budget, describeJob(job))
		}
		time.Sleep(interval)
	}
}

// followManagerLogs streams the live manager pod's log to $E2E_ARTIFACTS_DIR,
// so a manager gone by the time the spec fails still leaves its shutdown log.
// Best effort: nothing here can fail the spec.
func followManagerLogs(name string) {
	dir := os.Getenv("E2E_ARTIFACTS_DIR")
	if dir == "" {
		return
	}
	var pods corev1.PodList
	if err := k8sClient.List(context.Background(), &pods, client.InNamespace(e2eNamespace),
		client.MatchingLabels{"deployment": name}); err != nil {
		GinkgoWriter.Printf("not streaming manager logs: %v\n", err)
		return
	}
	pod := ""
	for _, p := range pods.Items {
		if p.DeletionTimestamp == nil {
			pod = p.Name
		}
	}
	if pod == "" {
		GinkgoWriter.Printf("not streaming manager logs: no live pod for %s\n", name)
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		GinkgoWriter.Printf("not streaming %s logs: %v\n", pod, err)
		return
	}
	f, err := os.Create(filepath.Join(dir, pod+".log"))
	if err != nil {
		GinkgoWriter.Printf("not streaming %s logs: %v\n", pod, err)
		return
	}
	cmd := exec.Command("kubectl", "logs", "-f", "--timestamps", "-n", e2eNamespace, pod)
	cmd.Stdout, cmd.Stderr = f, f
	if err := cmd.Start(); err != nil {
		GinkgoWriter.Printf("not streaming %s logs: %v\n", pod, err)
		_ = f.Close()
		return
	}
	DeferCleanup(func() {
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		// On failure a still-draining manager gets a minute to log its shutdown.
		grace := time.Duration(0)
		if CurrentSpecReport().Failed() {
			grace = time.Minute
		}
		select {
		case <-done:
		case <-time.After(grace):
			_ = cmd.Process.Kill()
			<-done
		}
		_ = f.Close()
	})
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
		followManagerLogs(name)
		Expect(k8sClient.Delete(ctx, &gitlabv1beta2.Runner{ObjectMeta: objMeta(name)})).To(Succeed())
		// The timeline prints only on failure; it is the evidence #136 lacks.
		var last string
		Eventually(func(g Gomega) {
			job, err := drainJob(pipeline)
			g.Expect(err).NotTo(HaveOccurred())
			if line := describeJob(job) + " | " + managerPodStates(name); line != last {
				GinkgoWriter.Printf("%s %s\n", time.Now().UTC().Format(time.TimeOnly), line)
				last = line
			}
			if job.Status == "running" {
				var cur gitlabv1beta2.Runner
				if err := k8sClient.Get(ctx, key(name), &cur); err != nil {
					StopTrying("the Runner was removed while its job was still running; " +
						settleDrainJob(pipeline, name, 3*time.Minute) + " | " + managerPodStates(name)).Wrap(err).Now()
				}
				if _, _, err := glab.Runners.GetRunnerDetails(got.Status.RunnerID); err != nil {
					StopTrying("the runner was deregistered from GitLab mid-job; " +
						settleDrainJob(pipeline, name, 3*time.Minute) + " | " + managerPodStates(name)).Wrap(err).Now()
				}
			}
			g.Expect(job.Status).To(Equal("success"))
		}, timeout, interval).Should(Succeed())
		waitGone(name)
	})
})
