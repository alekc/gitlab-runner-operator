package api

import (
	"errors"

	gitlabRunOp "go.alekc.dev/gitlab-runner-operator/api/v1alpha1"
)

type MockedGitlabClient struct {
	OnRegister func(config gitlabRunOp.RegisterNewRunnerOptions) (string, error)
}

func (m *MockedGitlabClient) Register(config gitlabRunOp.RegisterNewRunnerOptions) (string, error) {
	if m.OnRegister == nil {
		return "", errors.New("call is not defined")
	}
	return m.OnRegister(config)
}
