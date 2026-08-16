package generate

import (
	"regexp"
	"testing"

	k8stypes "k8s.io/apimachinery/pkg/types"
)

// upstreamSystemIDFormat is the regexp gitlab-runner applies when loading
// .runner_system_id; anything else is ignored and a fresh id is generated.
// See commands/internal/configfile/system_id_state.go.
var upstreamSystemIDFormat = regexp.MustCompile(`^[sr]_[0-9a-zA-Z]{12}$`)

func TestSystemID(t *testing.T) {
	const uid = k8stypes.UID("3f7c1e2a-1d4b-4a9e-9c8f-2b6d5e4a7c10")

	t.Run("matches the format gitlab-runner accepts", func(t *testing.T) {
		got := SystemID(uid)
		if !upstreamSystemIDFormat.MatchString(got) {
			t.Fatalf("SystemID(%q) = %q, does not match %s", uid, got, upstreamSystemIDFormat)
		}
	})

	// The whole point of the fix: the id must not change between reconciles, so
	// the manager keeps one identity in GitLab across pod replacement.
	t.Run("is stable for the same uid", func(t *testing.T) {
		if first, second := SystemID(uid), SystemID(uid); first != second {
			t.Fatalf("SystemID is not deterministic: %q then %q", first, second)
		}
	})

	t.Run("differs per uid", func(t *testing.T) {
		other := k8stypes.UID("9a1b2c3d-4e5f-6071-8293-a4b5c6d7e8f9")
		if SystemID(uid) == SystemID(other) {
			t.Fatalf("distinct uids produced the same system id %q", SystemID(uid))
		}
	})

	// Unreachable in practice, an object read back from the API server always
	// carries a uid. Pinned so the helper degrades to a well formed id rather
	// than something gitlab-runner would reject.
	t.Run("empty uid still yields a well formed id", func(t *testing.T) {
		if got := SystemID(""); !upstreamSystemIDFormat.MatchString(got) {
			t.Fatalf("SystemID(\"\") = %q, does not match %s", got, upstreamSystemIDFormat)
		}
	})
}
