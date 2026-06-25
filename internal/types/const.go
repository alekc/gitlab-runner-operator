package types

const ConfigVersionAnnotationKey = "config-version"
const ConfigMapKeyName = "config.toml"

// ConfigTokenKeyPrefix prefixes the per-entry authentication-token keys stored
// alongside config.toml in the rendered config Secret. The controller recovers
// a managed runner's token from "<prefix><entryName>" so the token never has to
// live in the CR status.
const ConfigTokenKeyPrefix = "authentication-token-"

// CACertMountDir is where a custom CA bundle is mounted into the runner pod and
// CACertFileName is the file it is projected to. CACertFile is the absolute
// path written into config.toml as tls-ca-file. The directory sits outside the
// config-Secret mount (/etc/gitlab-runner) to avoid nested mounts.
const (
	CACertMountDir = "/etc/gitlab-certs"
	CACertFileName = "ca.crt"
	CACertFile     = CACertMountDir + "/" + CACertFileName
)
