package api

import (
	"testing"

	"k8s.io/utils/pointer"

	"gitlabrunnerop.k8s.alekc.dev/api/v1alpha1"
)

func TestGitlabApi_Register(t *testing.T) {
	cl, _ := NewGitlabClient("9Bo36Uxwx6ay-cR-bCLh", "")
	_, _ = cl.Register(v1alpha1.RegisterNewRunnerOptions{
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
