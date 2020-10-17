package errors

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
)

// IsStale checks if the error is related to an update to the stale object.
// if yes, then that error will be suppressed
func IsStale(err error) bool {
	switch parsedError := err.(type) {
	case *errors.StatusError:
		errStatus := parsedError.Status()
		if errStatus.Code == 409 && strings.Contains(errStatus.Message, "please apply your changes to the latest version") {
			return true
		}
	}
	return false
}
