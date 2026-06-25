package v1beta2

import "fmt"

// DefaultCAKey is the key read from the referenced Secret or ConfigMap when
// CAKeyRef.Key is empty. It matches the Kubernetes convention used by TLS
// Secrets and the cluster root-CA ConfigMap.
const DefaultCAKey = "ca.crt"

// CASource references a PEM-encoded CA bundle used to verify the GitLab
// endpoint, both for the operator's own API calls and for the runner's
// connection. Exactly one of SecretKeyRef or ConfigMapKeyRef may be set.
type CASource struct {
	// SecretKeyRef selects a key in a Secret holding the PEM CA bundle.
	// +optional
	SecretKeyRef *CAKeyRef `json:"secretKeyRef,omitempty"`

	// ConfigMapKeyRef selects a key in a ConfigMap holding the PEM CA bundle.
	// +optional
	ConfigMapKeyRef *CAKeyRef `json:"configMapKeyRef,omitempty"`
}

// CAKeyRef points at a single key inside a Secret or ConfigMap.
type CAKeyRef struct {
	// Name of the Secret or ConfigMap.
	Name string `json:"name"`

	// Key holding the PEM CA bundle. Defaults to "ca.crt" when empty.
	// +optional
	Key string `json:"key,omitempty"`
}

// IsSet reports whether the source selects a CA bundle.
func (c *CASource) IsSet() bool {
	return c != nil && (c.SecretKeyRef != nil || c.ConfigMapKeyRef != nil)
}

// Validate enforces that at most one ref is set and that a set ref names a
// source. A nil source is valid (no custom CA).
func (c *CASource) Validate() error {
	if c == nil {
		return nil
	}
	if c.SecretKeyRef != nil && c.ConfigMapKeyRef != nil {
		return fmt.Errorf("caCertificate: set only one of secretKeyRef or configMapKeyRef")
	}
	if c.SecretKeyRef != nil && c.SecretKeyRef.Name == "" {
		return fmt.Errorf("caCertificate.secretKeyRef.name is required")
	}
	if c.ConfigMapKeyRef != nil && c.ConfigMapKeyRef.Name == "" {
		return fmt.Errorf("caCertificate.configMapKeyRef.name is required")
	}
	return nil
}
