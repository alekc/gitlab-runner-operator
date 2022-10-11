package generate

import (
	"bytes"
	"math"

	"github.com/BurntSushi/toml"
	"gitlab.k8s.alekc.dev/api/v1beta1"
	"gitlab.k8s.alekc.dev/config"
	"gitlab.k8s.alekc.dev/internal/crypto"
	"gitlab.k8s.alekc.dev/internal/types"
)

func TomlConfig(runner types.RunnerInfo) (gitlabConfig, configHashKey string, err error) {
	// ugly as hell, but its the best I can do for now to avoid the import loop.
	// Blame the kubebuilder which cannot generate deepCopy for external workspace
	switch r := runner.(type) {
	case *v1beta1.Runner:
		return SingleRunnerConfig(r)
	}
	panic("unknown runner type")
}
func SingleRunnerConfig(r *v1beta1.Runner) (gitlabConfig, configHashKey string, err error) {
	// define sensible config for some configuration values
	runnerConfig := &config.RunnerConfig{
		Name:  r.Name,
		Limit: 10,
		RunnerCredentials: config.RunnerCredentials{
			Token: r.Status.AuthenticationToken,
			URL:   r.Spec.GitlabInstanceURL,
		},
		RunnerSettings: config.RunnerSettings{
			Environment: r.Spec.Environment,
			Executor:    "kubernetes",
			Kubernetes:  &r.Spec.ExecutorConfig,
		},
	}
	// set the namespace to the same one as the runner object if not declared otherwise
	if runnerConfig.RunnerSettings.Kubernetes.Namespace == "" {
		runnerConfig.RunnerSettings.Kubernetes.Namespace = r.Namespace
	}
	rootConfig := &config.Config{
		ListenAddress: ":9090",
		Concurrent:    int(math.Max(1, float64(r.Spec.Concurrent))),
		CheckInterval: int(math.Max(3, float64(r.Spec.CheckInterval))),
		LogLevel:      r.Spec.LogLevel,
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

//
// // MultiRunnerConfig initialize config for multiple runners object
// func MultiRunnerConfig(runnerObject *v1beta1.MultiRunner) (gitlabConfig, configHashKey string, err error) {
// 	// create configuration for the runners
// 	var runners []*config.RunnerConfig
// 	for _, entry := range runnerObject.Spec.Entries {
// 		// define sensible config for some configuration values
// 		runnerConfig := config.RunnerConfig{
// 			Name:  entry.Name,
// 			Limit: 10,
// 			RunnerCredentials: config.RunnerCredentials{
// 				Token: runnerObject.Status.AuthTokens[*entry.RegistrationConfig.Token],
// 				URL:   runnerObject.Spec.GitlabInstanceURL,
// 			},
// 			RunnerSettings: config.RunnerSettings{
// 				Environment: entry.Environment,
// 				Executor:    "kubernetes",
// 				Kubernetes:  &entry.ExecutorConfig,
// 			},
// 		}
// 		// set the namespace to the same one as the runner object if not declared otherwise
// 		if runnerConfig.RunnerSettings.Kubernetes.Namespace == "" {
// 			runnerConfig.RunnerSettings.Kubernetes.Namespace = runnerObject.Namespace
// 		}
// 		runners = append(runners, &runnerConfig)
// 	}
//
// 	// begin to construct the default value
// 	rootConfig := &config.Config{
// 		ListenAddress: ":9090",
// 		Concurrent:    int(math.Max(1, float64(runnerObject.Spec.Concurrent))),
// 		CheckInterval: int(math.Max(3, float64(runnerObject.Spec.CheckInterval))),
// 		LogLevel:      runnerObject.Spec.LogLevel,
// 		Runners:       runners,
// 	}
//
// 	// if not explicit, define the log level
// 	if rootConfig.LogLevel == "" {
// 		rootConfig.LogLevel = "info"
// 	}
//
// 	var buff bytes.Buffer
// 	tomlEncoder := toml.NewEncoder(&buff)
// 	err = tomlEncoder.Encode(rootConfig)
// 	if err != nil {
// 		return "", "", err
// 	}
//
// 	gitlabConfig = buff.String()
// 	configHashKey = crypto.StringToSHA1(gitlabConfig)
// 	return buff.String(), configHashKey, nil
// }
