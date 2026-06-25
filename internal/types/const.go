package types

const ConfigVersionAnnotationKey = "config-version"
const ConfigMapKeyName = "config.toml"

// ConfigTokenKeyPrefix prefixes the per-entry authentication-token keys stored
// alongside config.toml in the rendered config Secret. The controller recovers
// a managed runner's token from "<prefix><entryName>" so the token never has to
// live in the CR status.
const ConfigTokenKeyPrefix = "authentication-token-"

// CACertFileName is the config-Secret data key (and projected filename) for a
// custom CA bundle; CACertFile is its absolute path inside the runner container,
// written into config.toml as tls-ca-file. The CA is stored in the config Secret
// alongside config.toml, which is mounted at /etc/gitlab-runner, so no extra
// volume is needed. Keep this prefix in sync with the config volume mount path
// in validate.Deployment.
const (
	CACertFileName = "ca.crt"
	CACertFile     = "/etc/gitlab-runner/" + CACertFileName
)
