package generate

import (
	"bytes"
	"math"

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
	runnerConfig := &config.RunnerConfig{
		Name:  runnerObject.Name,
		Limit: 10,
		RunnerCredentials: config.RunnerCredentials{
			Token: runnerObject.Status.AuthenticationToken,
			URL:   instanceUrl,
		},
		RunnerSettings: config.RunnerSettings{
			Executor:   "kubernetes",
			Kubernetes: &runnerObject.Spec.ExecutorConfig,
		},
	}
	// set the namespace to the same one as the runner object if not declared otherwise
	if runnerConfig.RunnerSettings.Kubernetes.Namespace == "" {
		runnerConfig.RunnerSettings.Kubernetes.Namespace = runnerObject.Namespace
	}
	rootConfig := &config.Config{
		ListenAddress: ":9090",
		Concurrent:    int(math.Max(1, float64(runnerObject.Spec.Concurrent))),
		LogLevel:      runnerObject.Spec.LogLevel,
		Runners:       []*config.RunnerConfig{runnerConfig},
	}

	// if not explicit, define the log level
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
