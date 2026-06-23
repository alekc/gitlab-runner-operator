package types

const ConfigVersionAnnotationKey = "config-version"
const ConfigMapKeyName = "config.toml"

// ConfigTokenKeyPrefix prefixes the per-entry authentication-token keys stored
// alongside config.toml in the rendered config Secret. The controller recovers
// a managed runner's token from "<prefix><entryName>" so the token never has to
// live in the CR status.
const ConfigTokenKeyPrefix = "authentication-token-"
