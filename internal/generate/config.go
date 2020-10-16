package generate

import (
	"bytes"

	"github.com/BurntSushi/toml"
	gitlabRunOp "go.alekc.dev/gitlab-runner-operator/api/v1alpha1"
	"go.alekc.dev/gitlab-runner-operator/config"
)

func ConfigText(runnerObject *gitlabRunOp.Runner) (string, error) {
	// define sensible config for some of the configuration values
	instanceUrl := runnerObject.Spec.GitlabInstanceURL
	if instanceUrl == "" {
		instanceUrl = "https://gitlab.com/"
	}
	rootConfig := &config.Config{
		ListenAddress: ":9090",
		Concurrent:    1,
		LogLevel:      runnerObject.Spec.LogLevel,
		Runners: []*config.RunnerConfig{{
			Name:  "test-runnerObject",
			Limit: 10,
			RunnerCredentials: config.RunnerCredentials{
				Token: runnerObject.Status.AuthenticationToken,
				URL:   instanceUrl,
			},
			RunnerSettings: config.RunnerSettings{
				Executor:   "kubernetes",
				Kubernetes: &runnerObject.Spec.Config,
			},
		}},
	}
	if rootConfig.LogLevel == "" {
		rootConfig.LogLevel = "info"
	}

	var buff bytes.Buffer
	tomlEncoder := toml.NewEncoder(&buff)
	err := tomlEncoder.Encode(rootConfig)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}
