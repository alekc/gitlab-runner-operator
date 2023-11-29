package api

import (
	"errors"
	"github.com/xanzy/go-gitlab"
)

type MockedGitlabClient struct {
	//OnRegister       func(config v1beta2.RegisterNewRunnerOptions) (string, error)
	OnDeleteByTokens func(token string) (*gitlab.Response, error)
}

//func (m *MockedGitlabClient) Register(config v1beta2.RegisterNewRunnerOptions) (string, error) {
//	if m.OnRegister == nil {
//		return "", errors.New("call is not defined")
//	}
//	return m.OnRegister(config)
//}

func (m *MockedGitlabClient) DeleteByToken(token string) (*gitlab.Response, error) {
	if m.OnDeleteByTokens == nil {
		return &gitlab.Response{}, errors.New("call is not defined")
	}
	return m.OnDeleteByTokens(token)
}
