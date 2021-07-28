package api

import (
	"errors"

	"gitlab.k8s.alekc.dev/api/v1beta1"
)

type MockedGitlabClient struct {
	OnRegister func(config v1beta1.RegisterNewRunnerOptions) (string, error)
}

func (m *MockedGitlabClient) Register(config v1beta1.RegisterNewRunnerOptions) (string, error) {
	if m.OnRegister == nil {
		return "", errors.New("call is not defined")
	}
	return m.OnRegister(config)
}
