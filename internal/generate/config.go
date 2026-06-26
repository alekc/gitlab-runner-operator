package generate

import (
	"bytes"
	"math"

	"github.com/BurntSushi/toml"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/config"
	"gitlab.k8s.alekc.dev/internal/crypto"
	"gitlab.k8s.alekc.dev/internal/types"
)

// TomlConfig renders the gitlab-runner config.toml for the object. Tokens are
// the resolved authentication tokens keyed by runner/entry name (the token is
// never read from status; the caller supplies it from the create/refresh result
// or the existing config Secret).
func TomlConfig(runner types.RunnerInfo, tokens map[string]string, caPEM []byte) (gitlabConfig, configHashKey string, err error) {
	// ugly as hell, but its the best I can do for now to avoid the import loop.
	// Blame the kubebuilder which cannot generate deepCopy for external workspace
	switch r := runner.(type) {
	case *v1beta2.Runner:
		return SingleRunnerConfig(r, tokens, caPEM)
	case *v1beta2.MultiRunner:
		return MultiRunnerConfig(r, tokens, caPEM)
	}
	panic("unknown runner type")
}
func SingleRunnerConfig(r *v1beta2.Runner, tokens map[string]string, caPEM []byte) (gitlabConfig, configHashKey string, err error) {
	// define sensible config for some configuration values
	runnerConfig := &config.RunnerConfig{
		Name:  r.Name,
		Limit: 10,
		RunnerCredentials: config.RunnerCredentials{
			Token: tokens[r.Name],
			URL:   r.Spec.GitlabInstanceURL,
		},
		RunnerSettings: config.RunnerSettings{
			Environment: r.Spec.Environment,
			Executor:    "kubernetes",
			Kubernetes:  &r.Spec.ExecutorConfig,
		},
	}
	// resolve the executor namespace via the shared defaulting rule so it
	// matches the namespace RBAC was provisioned for (crud.BuildNamespaces)
	runnerConfig.RunnerSettings.Kubernetes.Namespace = r.Spec.ExecutorConfig.EffectiveNamespace(r.Namespace)
	// point the runner at the mounted custom CA when one is configured
	if r.Spec.CACertificate.IsSet() {
		runnerConfig.RunnerCredentials.TLSCAFile = types.CACertFile
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
	// Fold the CA bundle into the hash so a CA content change (rotation) bumps
	// the config version and rolls the deployment, even though config.toml only
	// references tls-ca-file by path. Empty caPEM keeps the no-CA hash unchanged.
	hashInput := gitlabConfig
	if len(caPEM) > 0 {
		hashInput += "\n" + string(caPEM)
	}
	configHashKey = crypto.StringToSHA1(hashInput)
	return buff.String(), configHashKey, nil
}

// MultiRunnerConfig initialize config for multiple runners object
func MultiRunnerConfig(runnerObject *v1beta2.MultiRunner, tokens map[string]string, caPEM []byte) (gitlabConfig, configHashKey string, err error) {
	// create configuration for the runners
	var runners []*config.RunnerConfig
	for _, entry := range runnerObject.Spec.Entries {
		// executorConfig is a separate variable due to go's loop bug
		executorConfig := entry.ExecutorConfig
		runnerConfig := config.RunnerConfig{
			Name:  entry.Name,
			Limit: 10,
			RunnerCredentials: config.RunnerCredentials{
				Token: tokens[entry.Name],
				URL:   runnerObject.Spec.GitlabInstanceURL,
			},
			RunnerSettings: config.RunnerSettings{
				Environment: entry.Environment,
				Executor:    "kubernetes",
				Kubernetes:  &executorConfig,
			},
		}
		// resolve the executor namespace via the shared defaulting rule so it
		// matches the namespace RBAC was provisioned for (crud.BuildNamespaces)
		runnerConfig.RunnerSettings.Kubernetes.Namespace = executorConfig.EffectiveNamespace(runnerObject.Namespace)
		// point each runner at the mounted custom CA when one is configured
		if runnerObject.Spec.CACertificate.IsSet() {
			runnerConfig.RunnerCredentials.TLSCAFile = types.CACertFile
		}
		runners = append(runners, &runnerConfig)
	}

	// begin to construct the default value
	rootConfig := &config.Config{
		ListenAddress: ":9090",
		Concurrent:    int(math.Max(1, float64(runnerObject.Spec.Concurrent))),
		CheckInterval: int(math.Max(3, float64(runnerObject.Spec.CheckInterval))),
		LogLevel:      runnerObject.Spec.LogLevel,
		Runners:       runners,
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
	// Fold the CA bundle into the hash so a CA content change (rotation) bumps
	// the config version and rolls the deployment, even though config.toml only
	// references tls-ca-file by path. Empty caPEM keeps the no-CA hash unchanged.
	hashInput := gitlabConfig
	if len(caPEM) > 0 {
		hashInput += "\n" + string(caPEM)
	}
	configHashKey = crypto.StringToSHA1(hashInput)
	return buff.String(), configHashKey, nil
}
