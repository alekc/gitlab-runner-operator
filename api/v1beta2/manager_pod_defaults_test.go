package v1beta2

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

// The reconcile compares the manager pod's image pull policy and security
// context against the live pod template. Container defaulting DOES apply to a
// Deployment template, so the only reason that compare converges is that these
// accessors never return a zero value for the apiserver to fill in. An accessor
// that starts returning "" or nil makes the field permanently unsettleable.
func TestManagerPodAccessors_NeverReturnZero(t *testing.T) {
	runner := &Runner{}
	multi := &MultiRunner{}

	for name, got := range map[string]corev1.PullPolicy{
		"Runner":      runner.RunnerImagePullPolicy(),
		"MultiRunner": multi.RunnerImagePullPolicy(),
	} {
		if got == "" {
			t.Errorf("%s.RunnerImagePullPolicy() is empty; the apiserver would default it", name)
		}
	}

	for name, got := range map[string]*corev1.SecurityContext{
		"Runner":      runner.RunnerSecurityContext(),
		"MultiRunner": multi.RunnerSecurityContext(),
	} {
		if got == nil {
			t.Errorf("%s.RunnerSecurityContext() is nil; the compare would never settle", name)
		}
	}

	for name, got := range map[string]string{
		"Runner":      runner.RunnerImage(),
		"MultiRunner": multi.RunnerImage(),
	} {
		if got == "" {
			t.Errorf("%s.RunnerImage() is empty; the pod would have no image", name)
		}
	}
}
