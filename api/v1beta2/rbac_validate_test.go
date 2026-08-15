package v1beta2

import "testing"

// TestBuildNamespaceAllowed checks the reconciler's executor build-namespace
// allow-list: an empty value or the runner's own namespace is always allowed;
// any other namespace needs the operator allow-list or the "*" wildcard.
func TestBuildNamespaceAllowed(t *testing.T) {
	const own = "runner-ns"
	cases := []struct {
		name    string
		ns      string
		allowed []string
		want    bool
	}{
		{name: "empty ok", ns: "", want: true},
		{name: "own namespace ok", ns: own, want: true},
		{name: "cross namespace rejected by default", ns: "build", want: false},
		{name: "cross namespace allowed when listed", ns: "build", allowed: []string{"build"}, want: true},
		{name: "cross namespace allowed via wildcard", ns: "kube-system", allowed: []string{"*"}, want: true},
		{name: "cross namespace not in list rejected", ns: "kube-system", allowed: []string{"build"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BuildNamespaceAllowed(tc.ns, own, tc.allowed); got != tc.want {
				t.Fatalf("BuildNamespaceAllowed(%q, %q, %v) = %v, want %v", tc.ns, own, tc.allowed, got, tc.want)
			}
		})
	}
}
