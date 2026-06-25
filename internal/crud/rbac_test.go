package crud

import (
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta2"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildNamespaces(t *testing.T) {
	t.Run("runner defaults to its own namespace", func(t *testing.T) {
		r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
		assertSet(t, BuildNamespaces(r), []string{"rns"})
	})
	t.Run("runner honours executor namespace", func(t *testing.T) {
		r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
		r.Spec.ExecutorConfig.Namespace = "build"
		assertSet(t, BuildNamespaces(r), []string{"build"})
	})
	t.Run("multirunner unions distinct entry namespaces", func(t *testing.T) {
		mr := &v1beta2.MultiRunner{ObjectMeta: metav1.ObjectMeta{Name: "m", Namespace: "rns"}}
		e1 := v1beta2.MultiRunnerEntry{Name: "a"} // defaults to rns
		e2 := v1beta2.MultiRunnerEntry{Name: "b"}
		e2.ExecutorConfig.Namespace = "build"
		e3 := v1beta2.MultiRunnerEntry{Name: "c"}
		e3.ExecutorConfig.Namespace = "build" // duplicate, must collapse
		mr.Spec.Entries = []v1beta2.MultiRunnerEntry{e1, e2, e3}
		assertSet(t, BuildNamespaces(mr), []string{"rns", "build"})
	})
}

func TestDesiredRoleRules(t *testing.T) {
	rules := desiredRoleRules()
	want := map[string][]string{
		"pods":            {"get", "list", "watch", "create", "delete"},
		"pods/exec":       {"get", "create", "patch", "delete"},
		"pods/attach":     {"get", "create", "patch", "delete"},
		"pods/log":        {"get", "list"},
		"secrets":         {"get", "create", "update", "delete"},
		"services":        {"get", "create"},
		"serviceaccounts": {"get"},
	}
	for resource, wantVerbs := range want {
		verbs := verbsFor(rules, resource)
		if verbs == nil {
			t.Fatalf("no rule for resource %q", resource)
		}
		for _, v := range wantVerbs {
			if !contains(verbs, v) {
				t.Errorf("resource %q missing verb %q (got %v)", resource, v, verbs)
			}
		}
	}
	// least privilege: no wildcards anywhere
	for _, r := range rules {
		if contains(r.APIGroups, "*") || contains(r.Resources, "*") || contains(r.Verbs, "*") {
			t.Errorf("rule contains a wildcard: %+v", r)
		}
	}
}

func verbsFor(rules []rbacv1.PolicyRule, resource string) []string {
	for _, r := range rules {
		if contains(r.Resources, resource) {
			return r.Verbs
		}
	}
	return nil
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func assertSet(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	seen := map[string]struct{}{}
	for _, g := range got {
		seen[g] = struct{}{}
	}
	for _, w := range want {
		if _, ok := seen[w]; !ok {
			t.Fatalf("got %v, want %v (missing %q)", got, want, w)
		}
	}
}
