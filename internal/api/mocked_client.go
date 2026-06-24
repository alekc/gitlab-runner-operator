package api

import (
	"errors"

	"gitlab.k8s.alekc.dev/api/v1beta2"
)

// MockedGitlabClient is a test double for GitlabClient. Unset callbacks use
// permissive defaults (token valid) so tests that only exercise create/delete
// need not configure the verify hook.
type MockedGitlabClient struct {
	OnCreateRunner func(opts v1beta2.RunnerCreateOptions) (CreatedRunner, error)
	OnDeleteRunner func(token string) error
	OnVerifyToken  func(token string) (bool, error)
}

func (m *MockedGitlabClient) CreateRunner(opts v1beta2.RunnerCreateOptions) (CreatedRunner, error) {
	if m.OnCreateRunner == nil {
		return CreatedRunner{}, errors.New("call is not defined")
	}
	return m.OnCreateRunner(opts)
}

func (m *MockedGitlabClient) DeleteRunner(token string) error {
	if m.OnDeleteRunner == nil {
		return errors.New("call is not defined")
	}
	return m.OnDeleteRunner(token)
}

func (m *MockedGitlabClient) VerifyToken(token string) (bool, error) {
	if m.OnVerifyToken == nil {
		return true, nil
	}
	return m.OnVerifyToken(token)
}
