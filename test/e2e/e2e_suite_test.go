// Package e2e holds the live end-to-end suite. It runs against the cluster in
// the current kube context (a kind cluster in CI and locally) and a real GitLab
// instance. GITLAB_E2E_URL, GITLAB_E2E_TOKEN (api scope, Maintainer role) and
// GITLAB_E2E_PROJECT_ID are REQUIRED: the suite fails hard if they are missing
// or invalid, rather than skipping, so a misconfigured run is never a silent
// pass. `make test` excludes this package, so unit runs are unaffected.
package e2e

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	e2eNamespace = "default"
	// defaultJobTag is the RUNNER_TAG default in the test project's
	// .gitlab-ci.yml, used when GITLAB_E2E_RUNNER_TAG is unset.
	defaultJobTag = "test-gitlab-runner"
	timeout       = 6 * time.Minute
	// dispatchTimeout is the separate budget for GitLab to take build-job out
	// of "created". Queueing is GitLab's latency, not the runner's, and four
	// legs triggering pipelines seconds apart have taken minutes (#87).
	dispatchTimeout = 8 * time.Minute
	interval        = 5 * time.Second
)

// jobTag is the tag this run registers its runner with and pins its pipeline
// to. CI sets a per-run value so concurrent runs cannot pick up each other's
// jobs; a sibling doing so would kill the job when it tears its cluster down.
var jobTag = func() string {
	if t := os.Getenv("GITLAB_E2E_RUNNER_TAG"); t != "" {
		return t
	}
	return defaultJobTag
}()

var (
	k8sClient     client.Client
	glab          *gitlab.Client
	gitlabURL     string
	gitlabToken   string
	projectID     int
	defaultBranch string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gitlab-runner-operator e2e suite")
}

// requireEnv returns the value of key or fails the suite hard when it is empty.
func requireEnv(key string) string {
	v := os.Getenv(key)
	Expect(v).NotTo(BeEmpty(), key+" is required for the e2e suite (no skipping)")
	return v
}

var _ = BeforeSuite(func() {
	gitlabURL = requireEnv("GITLAB_E2E_URL")
	gitlabToken = requireEnv("GITLAB_E2E_TOKEN")

	var err error
	projectID, err = strconv.Atoi(requireEnv("GITLAB_E2E_PROJECT_ID"))
	Expect(err).NotTo(HaveOccurred(), "GITLAB_E2E_PROJECT_ID must be an integer")

	// Build the GitLab API client used by the specs to trigger pipelines, read
	// jobs, and mint/delete runners.
	glab, err = gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabURL))
	Expect(err).NotTo(HaveOccurred())

	// Validate the credentials hard: the token must be able to read the project
	// (api scope). A bad URL/token/project id fails here with a clear message.
	proj, _, err := glab.Projects.GetProject(projectID, nil)
	Expect(err).NotTo(HaveOccurred(),
		"GITLAB_E2E_* look invalid: cannot read project %d (need a token with api scope)", projectID)
	defaultBranch = proj.DefaultBranch
	Expect(defaultBranch).NotTo(BeEmpty(), "project has no default branch")

	cfg, err := ctrlconfig.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "a reachable kubeconfig is required (kind cluster)")

	Expect(gitlabv1beta2.AddToScheme(scheme.Scheme)).To(Succeed())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

// mintRunner registers a project runner directly via the GitLab API and returns
// its id and glrt- authentication token, for the bring-your-own-token specs.
func mintRunner(tags []string) (int, string) {
	GinkgoHelper()
	pid := int64(projectID)
	runner, _, err := glab.Users.CreateUserRunner(&gitlab.CreateUserRunnerOptions{
		RunnerType:  gitlab.Ptr("project_type"),
		ProjectID:   &pid,
		TagList:     &tags,
		RunUntagged: gitlab.Ptr(false),
	})
	Expect(err).NotTo(HaveOccurred(), "minting a runner via the API failed (token needs create_runner)")
	return int(runner.ID), runner.Token
}

// deleteRunnerByID best-effort removes a runner from GitLab (cleanup).
func deleteRunnerByID(id int) {
	if id == 0 {
		return
	}
	_, _ = glab.Runners.DeleteRegisteredRunnerByID(int64(id))
}

// deleteRunnerCR best-effort deletes a Runner CR (DeferCleanup).
func deleteRunnerCR(name string) {
	_ = k8sClient.Delete(context.Background(), &gitlabv1beta2.Runner{
		ObjectMeta: objMeta(name),
	})
}
