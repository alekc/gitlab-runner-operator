package api

import (
	"errors"

	"gitlab.k8s.alekc.dev/api/v1beta2"
)

// MockedGitlabClient is a test double for GitlabClient.
type MockedGitlabClient struct {
	OnCreateRunner func(opts v1beta2.RunnerCreateOptions) (CreatedRunner, error)
	OnDeleteRunner func(id int) error
}

func (m *MockedGitlabClient) CreateRunner(opts v1beta2.RunnerCreateOptions) (CreatedRunner, error) {
	if m.OnCreateRunner == nil {
		return CreatedRunner{}, errors.New("call is not defined")
	}
	return m.OnCreateRunner(opts)
}

func (m *MockedGitlabClient) DeleteRunner(id int) error {
	if m.OnDeleteRunner == nil {
		return errors.New("call is not defined")
	}
	return m.OnDeleteRunner(id)
}
