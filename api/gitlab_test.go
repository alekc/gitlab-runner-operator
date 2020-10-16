package api

import (
	"testing"

	"k8s.io/utils/pointer"

	gitlabRunOp "go.alekc.dev/gitlab-runner-operator/api/v1alpha1"
)

func TestGitlabApi_Register(t *testing.T) {
	cl, _ := NewGitlabClient("9Bo36Uxwx6ay-cR-bCLh", "")
	_, _ = cl.Register(gitlabRunOp.RegisterNewRunnerOptions{
		Token:          pointer.StringPtr("9Bo36Uxwx6ay-cR-bCLh"),
		Description:    nil,
		Info:           nil,
		Active:         nil,
		Locked:         nil,
		RunUntagged:    nil,
		TagList:        []string{"gitlab-testing-operator"},
		MaximumTimeout: nil,
	})
}
