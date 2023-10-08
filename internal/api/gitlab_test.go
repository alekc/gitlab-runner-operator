package api

import (
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta2"
	"k8s.io/utils/pointer"
)

func TestGitlabApi_Register(t *testing.T) {
	cl, _ := NewGitlabClient("9Bo36Uxwx6ay-cR-bCLh", "")
	_, _ = cl.Register(v1beta2.RegisterNewRunnerOptions{
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
