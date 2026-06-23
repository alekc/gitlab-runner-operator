package api

import (
	"errors"

	"gitlab.k8s.alekc.dev/api/v1beta2"
)

// MockedGitlabClient is a test double for GitlabClient. Unset callbacks use
// permissive defaults (token valid, runner exists) so tests that only exercise
// create/delete need not configure the verify/refresh/exists hooks.
type MockedGitlabClient struct {
	OnCreateRunner func(opts v1beta2.RunnerCreateOptions) (CreatedRunner, error)
	OnDeleteRunner func(id int) error
	OnVerifyToken  func(token string) (bool, error)
	OnRefreshToken func(id int) (CreatedRunner, error)
	OnRunnerExists func(id int) (bool, error)
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

func (m *MockedGitlabClient) VerifyToken(token string) (bool, error) {
	if m.OnVerifyToken == nil {
		return true, nil
	}
	return m.OnVerifyToken(token)
}

func (m *MockedGitlabClient) RefreshToken(id int) (CreatedRunner, error) {
	if m.OnRefreshToken == nil {
		return CreatedRunner{ID: id}, nil
	}
	return m.OnRefreshToken(id)
}

func (m *MockedGitlabClient) RunnerExists(id int) (bool, error) {
	if m.OnRunnerExists == nil {
		return true, nil
	}
	return m.OnRunnerExists(id)
}
