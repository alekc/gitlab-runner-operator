package api

import (
	"github.com/xanzy/go-gitlab"
	"gitlabrunnerop.k8s.alekc.dev/api/v1alpha1"
)

type GitlabClient interface {
	Register(config v1alpha1.RegisterNewRunnerOptions) (string, error)
}

type gitlabApi struct {
	gitlabApiClient *gitlab.Client
}

func (g *gitlabApi) Register(config v1alpha1.RegisterNewRunnerOptions) (string, error) {
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
	//if we have not passed any private gitlab url, then use a default one.
	if url == "" {
		url = "https://gitlab.com/"
	}

	//init the client
	obj := &gitlabApi{}
	obj.gitlabApiClient, err = gitlab.NewClient(token, gitlab.WithBaseURL(url))
	return obj, err
}
