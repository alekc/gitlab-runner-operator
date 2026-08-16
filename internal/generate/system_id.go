package generate

import (
	"crypto/sha256"
	"encoding/hex"

	k8stypes "k8s.io/apimachinery/pkg/types"
)

// systemIDLength is the id length after the prefix. gitlab-runner rejects a
// state file that does not match ^[sr]_[0-9a-zA-Z]{12}$.
const systemIDLength = 12

// SystemID derives a gitlab-runner system_id from the CR uid. Being a pure
// function, every replica and every restart recomputes the same value, which is
// what keeps the manager's identity stable in GitLab across pod replacement.
func SystemID(uid k8stypes.UID) string {
	sum := sha256.Sum256([]byte(uid))
	return "r_" + hex.EncodeToString(sum[:])[:systemIDLength]
}
