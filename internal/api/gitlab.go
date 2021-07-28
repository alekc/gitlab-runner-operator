package api

import (
	"github.com/xanzy/go-gitlab"
	"gitlab.k8s.alekc.dev/api/v1beta1"
)

type GitlabClient interface {
	Register(config v1beta1.RegisterNewRunnerOptions) (string, error)
}

type gitlabApi struct {
	gitlabApiClient *gitlab.Client
}

func (g *gitlabApi) Register(config v1beta1.RegisterNewRunnerOptions) (string, error) {
	// sadly we cannot do a direct conversion due to the presence of additional field
	convertedConfig := gitlab.RegisterNewRunnerOptions{
		Token:          config.Token,
		Description:    config.Description,
		Info:           (*gitlab.RegisterNewRunnerInfoOptions)(config.Info),
		Active:         config.Active,
		Locked:         config.Locked,
		RunUntagged:    config.RunUntagged,
		TagList:        config.TagList,
		MaximumTimeout: config.MaximumTimeout,
	}
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
