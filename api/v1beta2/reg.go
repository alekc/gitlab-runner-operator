package v1beta2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GitlabRegInfo carries the desired auth configuration for a single runner unit
// (a standalone Runner, or one entry of a MultiRunner) together with its current
// registration state read from status. The reconciler fills the state fields
// after creating a managed runner or resolving a bring-your-own token, then
// hands it back via StoreRunnerRegistration.
type GitlabRegInfo struct {
	// Name is the [[runners]] name written into config.toml.
	Name string

	// Auth is the desired authentication configuration for this unit.
	Auth GitlabAuth

	// GitlabUrl is the GitLab instance this runner talks to.
	GitlabUrl string

	// CACertificate, when set, is the custom CA bundle source used to verify
	// the GitLab endpoint for this unit's API and runner connection.
	CACertificate *CASource

	// RunnerID is GitLab's numeric id for a managed runner (0 if unmanaged or
	// not yet created).
	RunnerID int

	// AuthToken is the runner authentication token written into config.toml.
	AuthToken string

	// TokenExpiresAt is GitLab's expiry for a managed runner token, if any.
	TokenExpiresAt *metav1.Time

	// RegistrationHash is the hash of the create options that produced the
	// current managed runner. A mismatch with the desired hash forces a
	// recreate.
	RegistrationHash string
}
