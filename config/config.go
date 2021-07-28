package config

import "gitlab.k8s.alekc.dev/api/v1beta1"

//nolint:lll
type Config struct {
	ListenAddress string          `toml:"listen_address,omitempty" json:"listen_address,omitempty"`
	Concurrent    int             `toml:"concurrent" json:"concurrent"`
	CheckInterval int             `toml:"check_interval" json:"check_interval" description:"Define active checking interval of jobs"`
	LogLevel      string          `toml:"log_level" json:"log_level" description:"Define log level (one of: panic, fatal, error, warning, info, debug)"`
	Runners       []*RunnerConfig `toml:"runners" json:"runners"`
}

//nolint:lll
type RunnerConfig struct {
	Name               string `toml:"name" json:"name" short:"name" long:"description" env:"RUNNER_NAME" description:"Runner name"`
	Limit              int    `toml:"limit,omitzero" json:"limit,omitempty" long:"limit" env:"RUNNER_LIMIT" description:"Maximum number of builds processed by this runner"`
	OutputLimit        int    `toml:"output_limit,omitzero" json:"output_limit" long:"output-limit" env:"RUNNER_OUTPUT_LIMIT" description:"Maximum build trace size in kilobytes"`
	RequestConcurrency int    `toml:"request_concurrency,omitzero" long:"request-concurrency" env:"RUNNER_REQUEST_CONCURRENCY" description:"Maximum concurrency for job requests"`

	RunnerCredentials
	RunnerSettings
}

//nolint:lll
type RunnerCredentials struct {
	URL         string `toml:"url" json:"url" short:"u" long:"url" env:"CI_SERVER_URL" required:"true" description:"Runner URL"`
	Token       string `toml:"token" json:"token" short:"t" long:"token" env:"CI_SERVER_TOKEN" required:"true" description:"Runner token"`
	TLSCAFile   string `toml:"tls-ca-file,omitempty" json:"tls-ca-file" long:"tls-ca-file" env:"CI_SERVER_TLS_CA_FILE" description:"File containing the certificates to verify the peer when using HTTPS"`
	TLSCertFile string `toml:"tls-cert-file,omitempty" json:"tls-cert-file" long:"tls-cert-file" env:"CI_SERVER_TLS_CERT_FILE" description:"File containing certificate for TLS client auth when using HTTPS"`
	TLSKeyFile  string `toml:"tls-key-file,omitempty" json:"tls-key-file" long:"tls-key-file" env:"CI_SERVER_TLS_KEY_FILE" description:"File containing private key for TLS client auth when using HTTPS"`
}
type RunnerSettings struct {
	Executor  string `toml:"executor" json:"executor" long:"executor" env:"RUNNER_EXECUTOR" required:"true" description:"Select executor, eg. shell, docker, etc."`
	BuildsDir string `toml:"builds_dir,omitempty" json:"builds_dir" long:"builds-dir" env:"RUNNER_BUILDS_DIR" description:"Directory where builds are stored"`
	CacheDir  string `toml:"cache_dir,omitempty" json:"cache_dir" long:"cache-dir" env:"RUNNER_CACHE_DIR" description:"Directory where build cache is stored"`
	CloneURL  string `toml:"clone_url,omitempty" json:"clone_url" long:"clone-url" env:"CLONE_URL" description:"Overwrite the default URL used to clone or fetch the git ref"`

	Environment     []string `toml:"environment,omitempty" json:"environment" long:"env" env:"RUNNER_ENV" description:"Custom environment variables injected to build environment"`
	PreCloneScript  string   `toml:"pre_clone_script,omitempty" json:"pre_clone_script" long:"pre-clone-script" env:"RUNNER_PRE_CLONE_SCRIPT" description:"Runner-specific command script executed before code is pulled"`
	PreBuildScript  string   `toml:"pre_build_script,omitempty" json:"pre_build_script" long:"pre-build-script" env:"RUNNER_PRE_BUILD_SCRIPT" description:"Runner-specific command script executed after code is pulled, just before build executes"`
	PostBuildScript string   `toml:"post_build_script,omitempty" json:"post_build_script" long:"post-build-script" env:"RUNNER_POST_BUILD_SCRIPT" description:"Runner-specific command script executed after code is pulled and just after build executes"`

	DebugTraceDisabled bool `toml:"debug_trace_disabled,omitempty" json:"debug_trace_disabled" long:"debug-trace-disabled" env:"RUNNER_DEBUG_TRACE_DISABLED" description:"When set to true Runner will disable the possibility of using the CI_DEBUG_TRACE feature"`

	Kubernetes *v1beta1.KubernetesConfig `toml:"kubernetes,omitempty" json:"kubernetes" group:"kubernetes executor" namespace:"kubernetes"`
}
