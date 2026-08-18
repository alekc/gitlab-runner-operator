package crud

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta2"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
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
		// pod_disruption_budget errors the whole job without these.
		"poddisruptionbudgets": {"get", "create"},
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

// The apiserver's escalation check means the operator can only grant what it
// already holds, so every executor rule must also appear in the manager
// ClusterRole. crud.go states this in prose; this asserts it.
func TestExecutorRulesAreHeldByTheManager(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "config", "rbac", "role.yaml"))
	if err != nil {
		t.Fatalf("read manager role: %v", err)
	}
	// sigs.k8s.io/yaml converts to JSON, so json tags are what bind here.
	var role struct {
		Rules []struct {
			APIGroups     []string `json:"apiGroups"`
			Resources     []string `json:"resources"`
			Verbs         []string `json:"verbs"`
			ResourceNames []string `json:"resourceNames"`
		} `json:"rules"`
	}
	if err := yaml.Unmarshal(raw, &role); err != nil {
		t.Fatalf("parse manager role: %v", err)
	}
	held := func(group, resource, verb string) bool {
		for _, r := range role.Rules {
			if !contains(r.APIGroups, group) || !contains(r.Resources, resource) {
				continue
			}
			// A rule narrowed to named objects does not authorise the executor's
			// unrestricted rule, and the apiserver escalation check agrees.
			if len(r.ResourceNames) > 0 {
				continue
			}
			if contains(r.Verbs, verb) || contains(r.Verbs, "*") {
				return true
			}
		}
		return false
	}
	for _, rule := range desiredRoleRules() {
		for _, g := range rule.APIGroups {
			for _, res := range rule.Resources {
				for _, v := range rule.Verbs {
					if !held(g, res, v) {
						t.Errorf("executor rule %q/%q verb %q is not held by the manager role", g, res, v)
					}
				}
			}
		}
	}
}

// crudScheme is the minimal scheme the RBAC reconcile needs.
func crudScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := rbacv1.AddToScheme(s); err != nil {
		t.Fatalf("add rbac to scheme: %v", err)
	}
	return s
}

func pdbRule(rules []rbacv1.PolicyRule) *rbacv1.PolicyRule {
	for i := range rules {
		if contains(rules[i].Resources, "poddisruptionbudgets") {
			return &rules[i]
		}
	}
	return nil
}

// A fresh cluster gets the full rule set on create.
func TestEnsureExecutorClusterRoleCreates(t *testing.T) {
	s := crudScheme(t)
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	if err := ensureExecutorClusterRole(context.Background(), cl, cl, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got rbacv1.ClusterRole
	if err := cl.Get(context.Background(),
		client.ObjectKey{Name: executorClusterRoleName}, &got); err != nil {
		t.Fatalf("clusterrole not created: %v", err)
	}
	if pdbRule(got.Rules) == nil {
		t.Error("created clusterrole has no poddisruptionbudgets rule")
	}
}

// The upgrade path. An existing cluster already carries this ClusterRole without
// the newer rules, so the whole grant depends on the drift branch converging it.
// Without this, a new install would work and every existing one would silently
// keep the old rules.
func TestEnsureExecutorClusterRoleCorrectsDrift(t *testing.T) {
	s := crudScheme(t)
	stale := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorClusterRoleName},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"}},
		},
	}
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(stale).Build()

	if err := ensureExecutorClusterRole(context.Background(), cl, cl, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got rbacv1.ClusterRole
	if err := cl.Get(context.Background(),
		client.ObjectKey{Name: executorClusterRoleName}, &got); err != nil {
		t.Fatalf("get clusterrole: %v", err)
	}
	rule := pdbRule(got.Rules)
	if rule == nil {
		t.Fatal("drift was not corrected: no poddisruptionbudgets rule after reconcile")
	}
	if !contains(rule.APIGroups, "policy") {
		t.Errorf("wrong apiGroup: %v", rule.APIGroups)
	}
	for _, v := range []string{"get", "create"} {
		if !contains(rule.Verbs, v) {
			t.Errorf("missing verb %q (got %v)", v, rule.Verbs)
		}
	}
	// Convergence is to the full desired set, not an append.
	if len(got.Rules) != len(desiredRoleRules()) {
		t.Errorf("rule count %d, want %d", len(got.Rules), len(desiredRoleRules()))
	}
}

// Rules removed from desiredRoleRules must also reach existing clusters, which
// is how a grant would be revoked in a later operator release.
func TestEnsureExecutorClusterRoleRevokesExtraRules(t *testing.T) {
	s := crudScheme(t)
	extra := append(desiredRoleRules(), rbacv1.PolicyRule{
		APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"create"},
	})
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorClusterRoleName},
		Rules:      extra,
	}).Build()

	if err := ensureExecutorClusterRole(context.Background(), cl, cl, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got rbacv1.ClusterRole
	if err := cl.Get(context.Background(),
		client.ObjectKey{Name: executorClusterRoleName}, &got); err != nil {
		t.Fatalf("get clusterrole: %v", err)
	}
	for _, r := range got.Rules {
		if contains(r.Resources, "deployments") {
			t.Error("an extra rule survived the reconcile; revocation would not reach existing clusters")
		}
	}
}
