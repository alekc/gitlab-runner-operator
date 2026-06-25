package api

import (
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta2"
)

// TestGitlabApi_CreateRunner is a smoke test: it exercises the client wiring
// against the real endpoint with a throwaway token and ignores the result.
func TestGitlabApi_CreateRunner(t *testing.T) {
	cl, _ := NewGitlabClient("9Bo36Uxwx6ay-cR-bCLh", "")
	_, _ = cl.CreateRunner(v1beta2.RunnerCreateOptions{
		RunnerType: "instance_type",
		TagList:    []string{"gitlab-testing-operator"},
	})
}
