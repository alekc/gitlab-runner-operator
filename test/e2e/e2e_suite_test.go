// Package e2e holds the live end-to-end suite. It runs against the cluster in
// the current kube context (a kind cluster in CI) and a real GitLab instance.
// It is skipped unless the GITLAB_E2E_* environment is provided, so it is safe
// to run from `make test` contexts and on fork PRs without secrets.
package e2e

import (
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	k8sClient   client.Client
	gitlabURL   string
	gitlabToken string
	projectID   int
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gitlab-runner-operator e2e suite")
}

var _ = BeforeSuite(func() {
	gitlabURL = os.Getenv("GITLAB_E2E_URL")
	gitlabToken = os.Getenv("GITLAB_E2E_TOKEN")
	pid := os.Getenv("GITLAB_E2E_PROJECT_ID")
	if gitlabURL == "" || gitlabToken == "" || pid == "" {
		Skip("GITLAB_E2E_URL, GITLAB_E2E_TOKEN and GITLAB_E2E_PROJECT_ID must be set; skipping live e2e")
	}

	var err error
	projectID, err = strconv.Atoi(pid)
	Expect(err).NotTo(HaveOccurred(), "GITLAB_E2E_PROJECT_ID must be an integer")

	cfg, err := ctrlconfig.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "a reachable kubeconfig is required (kind cluster)")

	Expect(gitlabv1beta2.AddToScheme(scheme.Scheme)).To(Succeed())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})
