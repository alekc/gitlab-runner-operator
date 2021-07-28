package crypto

import (
	"crypto/sha1"
	"encoding/hex"
)

// StringToSHA1 encodes a string into a SHA1 hash.
func StringToSHA1(input string) string {
	h := sha1.New()
	h.Write([]byte(input))

	return hex.EncodeToString(h.Sum(nil))
}
