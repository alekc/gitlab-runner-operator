package api

import (
	"gitlab.k8s.alekc.dev/internal/data/pointer"
	"io"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.k8s.alekc.dev/api/v1beta1"
)

type GitlabClient interface {
	Register(config v1beta1.RegisterNewRunnerOptions) (string, error)
	DeleteByToken(token string) (*gitlab.Response, error)
}

type gitlabApi struct {
	gitlabApiClient *gitlab.Client
}

func (g *gitlabApi) DeleteByToken(token string) (*gitlab.Response, error) {
	return g.gitlabApiClient.Runners.DeleteRegisteredRunner(&gitlab.DeleteRegisteredRunnerOptions{
		Token: pointer.String(token),
	})
}
func (g *gitlabApi) Register(config v1beta1.RegisterNewRunnerOptions) (string, error) {
	// The SDK widened MaximumTimeout to *int64; convert from the CRD's *int.
	var maximumTimeout *int64
	if config.MaximumTimeout != nil {
		v := int64(*config.MaximumTimeout)
		maximumTimeout = &v
	}
	// sadly we cannot do a direct conversion due to the presence of additional field
	convertedConfig := gitlab.RegisterNewRunnerOptions{
		Token:          config.Token,
		Description:    config.Description,
		Info:           (*gitlab.RegisterNewRunnerInfoOptions)(config.Info),
		Active:         config.Active,
		Locked:         config.Locked,
		RunUntagged:    config.RunUntagged,
		TagList:        pointer.StringSlice(config.TagList),
		MaximumTimeout: maximumTimeout,
	}
	runner, resp, err := g.gitlabApiClient.Runners.RegisterNewRunner(&convertedConfig)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

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
