package generate

import (
	"bytes"
	"math"

	"github.com/BurntSushi/toml"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/config"
	"gitlab.k8s.alekc.dev/internal/crypto"
)

// ConfigText initialize default config object and returns it as a text
func ConfigText(runnerObject *v1beta2.Runner) (gitlabConfig, configHashKey string, err error) {
	// define sensible config for some configuration values
	runnerConfig := &config.RunnerConfig{
		Name:  runnerObject.Name,
		Limit: 10,
		RunnerCredentials: config.RunnerCredentials{
			Token: runnerObject.Status.AuthenticationToken,
			URL:   runnerObject.Spec.GitlabInstanceURL,
		},
		RunnerSettings: config.RunnerSettings{
			Environment: runnerObject.Spec.Environment,
			Executor:    "kubernetes",
			Kubernetes:  &runnerObject.Spec.ExecutorConfig,
		},
	}
	// set the namespace to the same one as the runner object if not declared otherwise
	if runnerConfig.RunnerSettings.Kubernetes.Namespace == "" {
		runnerConfig.RunnerSettings.Kubernetes.Namespace = runnerObject.Namespace
	}
	rootConfig := &config.Config{
		ListenAddress: ":9090",
		Concurrent:    int(math.Max(1, float64(runnerObject.Spec.Concurrent))),
		CheckInterval: int(math.Max(3, float64(runnerObject.Spec.CheckInterval))),
		LogLevel:      runnerObject.Spec.LogLevel,
		Runners:       []*config.RunnerConfig{runnerConfig},
	}

	// if not explicit, define the log level
	if rootConfig.LogLevel == "" {
		rootConfig.LogLevel = "info"
	}

	var buff bytes.Buffer
	tomlEncoder := toml.NewEncoder(&buff)
	err = tomlEncoder.Encode(rootConfig)
	if err != nil {
		return "", "", err
	}

	gitlabConfig = buff.String()
	configHashKey = crypto.StringToSHA1(gitlabConfig)
	return buff.String(), configHashKey, nil
}
