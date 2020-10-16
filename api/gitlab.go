package api

import (
	"github.com/xanzy/go-gitlab"
	gitlabRunOp "go.alekc.dev/gitlab-runner-operator/api/v1alpha1"
)

type GitlabClient interface {
	Register(config gitlabRunOp.RegisterNewRunnerOptions) (string, error)
}

type gitlabApi struct {
	gitlabApiClient *gitlab.Client
}

func (g *gitlabApi) Register(config gitlabRunOp.RegisterNewRunnerOptions) (string, error) {
	convertedConfig := gitlab.RegisterNewRunnerOptions(config)
	runner, resp, err := g.gitlabApiClient.Runners.RegisterNewRunner(&convertedConfig)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return runner.Token, nil
}
func NewGitlabClient(token, url string) (GitlabClient, error) {
	var err error
	// if we have not passed any private gitlab url, then use a default one.
	if url == "" {
		url = "https://gitlab.com/"
	}

	// init the client
	obj := &gitlabApi{}
	obj.gitlabApiClient, err = gitlab.NewClient(token, gitlab.WithBaseURL(url))
	return obj, err
}
