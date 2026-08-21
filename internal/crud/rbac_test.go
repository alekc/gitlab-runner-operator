package crud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta2"
	internalTypes "gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
		"configmaps":      {"get", "create", "delete"},
		"events":          {"list", "watch"},
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
	// The point of #64: an optional grant must not ride the always-on set.
	if pdbRule(rules) != nil {
		t.Error("poddisruptionbudgets is in the base rules; it belongs to the optional role")
	}
	// least privilege: no wildcards anywhere
	for _, r := range rules {
		if contains(r.APIGroups, "*") || contains(r.Resources, "*") || contains(r.Verbs, "*") {
			t.Errorf("rule contains a wildcard: %+v", r)
		}
	}
}

func TestDesiredPDBRules(t *testing.T) {
	rule := pdbRule(desiredPDBRules())
	if rule == nil {
		t.Fatal("no poddisruptionbudgets rule in the optional set")
	}
	if !contains(rule.APIGroups, "policy") {
		t.Errorf("wrong apiGroup: %v", rule.APIGroups)
	}
	for _, v := range []string{"get", "create"} {
		if !contains(rule.Verbs, v) {
			t.Errorf("missing verb %q (got %v)", v, rule.Verbs)
		}
	}
	if len(desiredPDBRules()) != 1 {
		t.Errorf("optional role carries %d rules, want exactly the PDB one", len(desiredPDBRules()))
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
	for _, rule := range append(desiredRoleRules(), desiredPDBRules()...) {
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

// rbacClient builds a fake client with the scheme CreateRBACIfMissing needs.
func rbacClient(t *testing.T, objs ...client.Object) client.WithWatch {
	t.Helper()
	s := crudScheme(t)
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	return fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()
}

// testRunner is a Runner with the optional flag off unless pdb is true.
func testRunner(pdb bool) *v1beta2.Runner {
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns", UID: "uid-1"}}
	if pdb {
		r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
	}
	return r
}

// reconcileRBAC drives the production entry point. Every ClusterRole assertion
// goes through here: a helper reachable only from tests can stop being what
// production calls, and nothing would notice.
func reconcileRBAC(t *testing.T, cl client.WithWatch, r *v1beta2.Runner) {
	t.Helper()
	if err := CreateRBACIfMissing(context.Background(), cl, cl, r, logr.Discard()); err != nil {
		t.Fatalf("CreateRBACIfMissing: %v", err)
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
func TestEnsureExecutorClusterRolesCreatesBoth(t *testing.T) {
	cl := rbacClient(t)
	reconcileRBAC(t, cl, testRunner(false))
	var base rbacv1.ClusterRole
	if err := cl.Get(context.Background(),
		client.ObjectKey{Name: executorClusterRoleName}, &base); err != nil {
		t.Fatalf("base clusterrole not created: %v", err)
	}
	if pdbRule(base.Rules) != nil {
		t.Error("base clusterrole still grants poddisruptionbudgets")
	}
	var optional rbacv1.ClusterRole
	if err := cl.Get(context.Background(),
		client.ObjectKey{Name: executorPDBClusterRoleName}, &optional); err != nil {
		t.Fatalf("optional clusterrole not created: %v", err)
	}
	if pdbRule(optional.Rules) == nil {
		t.Error("optional clusterrole has no poddisruptionbudgets rule")
	}
}

// Drift convergence, driven through CreateRBACIfMissing. A fresh install gets
// correct rules from the create branch, so only this covers an existing cluster
// whose role predates a rule change.
func TestEnsureExecutorClusterRoleCorrectsDrift(t *testing.T) {
	stale := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorClusterRoleName},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"}},
		},
	}
	cl := rbacClient(t, stale)
	reconcileRBAC(t, cl, testRunner(false))
	var got rbacv1.ClusterRole
	if err := cl.Get(context.Background(),
		client.ObjectKey{Name: executorClusterRoleName}, &got); err != nil {
		t.Fatalf("get clusterrole: %v", err)
	}
	if verbsFor(got.Rules, "secrets") == nil {
		t.Fatal("drift was not corrected: base rules missing after reconcile")
	}
	// Convergence is to the full desired set, not an append.
	if len(got.Rules) != len(desiredRoleRules()) {
		t.Errorf("rule count %d, want %d", len(got.Rules), len(desiredRoleRules()))
	}
}

// The #64 revocation, driven through CreateRBACIfMissing. e2e cannot cover it:
// CI builds a fresh cluster, where the role is created already correct, so the
// drift branch is the only path that reaches a pre-existing install.
func TestEnsureExecutorClusterRoleDropsPDBFromTheBaseRole(t *testing.T) {
	legacy := append(desiredRoleRules(), desiredPDBRules()...)
	cl := rbacClient(t, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorClusterRoleName},
		Rules:      legacy,
	})
	reconcileRBAC(t, cl, testRunner(false))
	var got rbacv1.ClusterRole
	if err := cl.Get(context.Background(),
		client.ObjectKey{Name: executorClusterRoleName}, &got); err != nil {
		t.Fatalf("get clusterrole: %v", err)
	}
	if pdbRule(got.Rules) != nil {
		t.Error("the unconditional poddisruptionbudgets grant survived the upgrade")
	}
}

// Rules removed from desiredRoleRules must also reach existing clusters, which
// is how a grant would be revoked in a later operator release.
func TestEnsureExecutorClusterRoleRevokesExtraRules(t *testing.T) {
	extra := append(desiredRoleRules(), rbacv1.PolicyRule{
		APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"create"},
	})
	cl := rbacClient(t, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorClusterRoleName},
		Rules:      extra,
	})
	reconcileRBAC(t, cl, testRunner(false))
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

func pdbTrue() *bool  { b := true; return &b }
func pdbFalse() *bool { b := false; return &b }

// bindingNames renders BindingKeys as "namespace/name" for set comparison.
func bindingNames(keys []BindingKey) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k.Namespace+"/"+k.Name)
	}
	return out
}

func TestPDBNamespaces(t *testing.T) {
	t.Run("unset grants nothing", func(t *testing.T) {
		r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
		if got := pdbNamespaces(r); len(got) != 0 {
			t.Errorf("got %v, want none: upstream defaults the flag to false", got)
		}
	})
	t.Run("explicit false grants nothing", func(t *testing.T) {
		r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
		r.Spec.ExecutorConfig.PodDisruptionBudget = pdbFalse()
		if got := pdbNamespaces(r); len(got) != 0 {
			t.Errorf("got %v, want none", got)
		}
	})
	t.Run("true grants the effective namespace", func(t *testing.T) {
		r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
		r.Spec.ExecutorConfig.Namespace = "build"
		r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
		got := pdbNamespaces(r)
		if _, ok := got["build"]; !ok || len(got) != 1 {
			t.Errorf("got %v, want only build", got)
		}
	})
	// The refinement over a global OR: one entry must not widen the grant into
	// the namespaces its siblings target.
	t.Run("multirunner keeps namespaces independent", func(t *testing.T) {
		mr := &v1beta2.MultiRunner{ObjectMeta: metav1.ObjectMeta{Name: "m", Namespace: "rns"}}
		on := v1beta2.MultiRunnerEntry{Name: "on"}
		on.ExecutorConfig.Namespace = "a"
		on.ExecutorConfig.PodDisruptionBudget = pdbTrue()
		off := v1beta2.MultiRunnerEntry{Name: "off"}
		off.ExecutorConfig.Namespace = "b"
		mr.Spec.Entries = []v1beta2.MultiRunnerEntry{on, off}
		got := pdbNamespaces(mr)
		if _, ok := got["a"]; !ok {
			t.Error("namespace a should be granted")
		}
		if _, ok := got["b"]; ok {
			t.Error("namespace b must not inherit the grant from a sibling entry")
		}
	})
}

func TestDesiredBindings(t *testing.T) {
	t.Run("base only when the flag is unset", func(t *testing.T) {
		r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
		assertSet(t, bindingNames(DesiredBindings(r, []string{"rns"})), []string{"rns/" + r.ChildName()})
	})
	t.Run("adds the optional binding when set", func(t *testing.T) {
		r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
		r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
		assertSet(t, bindingNames(DesiredBindings(r, []string{"rns"})),
			[]string{"rns/" + r.ChildName(), "rns/" + pdbBindingPrefix + r.ChildName()})
	})
	t.Run("the two bindings reference different clusterroles", func(t *testing.T) {
		r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
		r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
		seen := map[string]string{}
		for _, k := range DesiredBindings(r, []string{"rns"}) {
			seen[k.Name] = k.ClusterRole
		}
		if seen[r.ChildName()] != executorClusterRoleName {
			t.Errorf("base binding references %q", seen[r.ChildName()])
		}
		if seen[pdbBindingPrefix+r.ChildName()] != executorPDBClusterRoleName {
			t.Errorf("optional binding references %q", seen[pdbBindingPrefix+r.ChildName()])
		}
	})
}

// managedBinding builds a labelled RoleBinding the way the operator would, so
// the prune's label selector matches it.
func managedBinding(obj internalTypes.RunnerInfo, namespace, name string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: rbacLabels(obj)},
	}
}

func bindingExists(t *testing.T, cl client.Client, namespace, name string) bool {
	t.Helper()
	var got rbacv1.RoleBinding
	err := cl.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, &got)
	if err != nil && !errors.IsNotFound(err) {
		t.Fatalf("get rolebinding %s/%s: %v", namespace, name, err)
	}
	return err == nil
}

// The defect #64 is really about. A namespace-keyed prune cannot see this,
// because the namespace is still in use by the base binding.
func TestDeleteRBACExceptRevokesTheOptionalBindingOnSpecFlip(t *testing.T) {
	s := crudScheme(t)
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns", UID: "uid-1"}}
	base, optional := r.ChildName(), pdbBindingPrefix+r.ChildName()
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(
		managedBinding(r, "rns", base),
		managedBinding(r, "rns", optional),
	).Build()

	// The flag is now unset, so DesiredBindings omits the optional binding.
	keep := DesiredBindings(r, []string{"rns"})
	if err := DeleteRBACExcept(context.Background(), cl, cl, r, keep, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bindingExists(t, cl, "rns", base) {
		t.Error("the base binding was pruned; the runner lost its executor RBAC")
	}
	if bindingExists(t, cl, "rns", optional) {
		t.Error("the optional binding survived a spec flip to off; the grant was never revoked")
	}
}

func TestDeleteRBACExceptKeepsTheOptionalBindingWhileEnabled(t *testing.T) {
	s := crudScheme(t)
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns", UID: "uid-1"}}
	r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
	optional := pdbBindingPrefix + r.ChildName()
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(
		managedBinding(r, "rns", r.ChildName()),
		managedBinding(r, "rns", optional),
	).Build()

	keep := DesiredBindings(r, []string{"rns"})
	if err := DeleteRBACExcept(context.Background(), cl, cl, r, keep, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bindingExists(t, cl, "rns", optional) {
		t.Error("the optional binding was pruned while the flag is still set")
	}
}

// The pre-existing behaviour must survive the rework: dropping a namespace from
// the spec prunes everything the runner had there, optional binding included.
func TestDeleteRBACExceptPrunesARemovedNamespace(t *testing.T) {
	s := crudScheme(t)
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns", UID: "uid-1"}}
	r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(
		managedBinding(r, "rns", r.ChildName()),
		managedBinding(r, "gone", r.ChildName()),
		managedBinding(r, "gone", pdbBindingPrefix+r.ChildName()),
	).Build()

	keep := DesiredBindings(r, []string{"rns"})
	if err := DeleteRBACExcept(context.Background(), cl, cl, r, keep, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bindingExists(t, cl, "gone", r.ChildName()) {
		t.Error("base binding in the removed namespace survived")
	}
	if bindingExists(t, cl, "gone", pdbBindingPrefix+r.ChildName()) {
		t.Error("optional binding in the removed namespace survived")
	}
	if !bindingExists(t, cl, "rns", r.ChildName()) {
		t.Error("binding in a kept namespace was pruned")
	}
}

// The finalizer path keeps both same-namespace bindings regardless of the flag,
// because owner references collect them, and prunes cross-namespace ones.
func TestDeleteRBACExceptFinalizerKeepsOwnNamespace(t *testing.T) {
	s := crudScheme(t)
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns", UID: "uid-1"}}
	optional := pdbBindingPrefix + r.ChildName()
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(
		managedBinding(r, "rns", r.ChildName()),
		managedBinding(r, "rns", optional),
		managedBinding(r, "build", r.ChildName()),
		managedBinding(r, "build", optional),
	).Build()

	keep := AllBindingsIn(r, r.GetNamespace())
	if err := DeleteRBACExcept(context.Background(), cl, cl, r, keep, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{r.ChildName(), optional} {
		if !bindingExists(t, cl, "rns", name) {
			t.Errorf("own-namespace binding %q was pruned; owner-reference GC should handle it", name)
		}
		if bindingExists(t, cl, "build", name) {
			t.Errorf("cross-namespace binding %q survived finalization and is orphaned", name)
		}
	}
}

// The grant path, end to end through CreateRBACIfMissing. Without this the
// whole feature can be a no-op and every other test still passes: the predicate
// and the prune are covered, but nothing asserts a binding is ever created.
func TestCreateRBACIfMissingGrantsTheOptionalBinding(t *testing.T) {
	s := crudScheme(t)
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("add corev1: %v", err)
	}
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns", UID: "uid-1"}}
	r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
	cl := fake.NewClientBuilder().WithScheme(s).Build()

	if err := CreateRBACIfMissing(context.Background(), cl, cl, r, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var optional rbacv1.RoleBinding
	if err := cl.Get(context.Background(), client.ObjectKey{
		Namespace: "rns", Name: pdbBindingPrefix + r.ChildName(),
	}, &optional); err != nil {
		t.Fatalf("optional binding was never created: %v", err)
	}
	// A binding pointing at the base role would grant nothing, and would look
	// correct to any test that only checks the object exists.
	if optional.RoleRef.Name != executorPDBClusterRoleName {
		t.Errorf("optional binding references %q, want %q", optional.RoleRef.Name, executorPDBClusterRoleName)
	}
	if optional.RoleRef.Kind != "ClusterRole" {
		t.Errorf("optional binding roleRef kind %q", optional.RoleRef.Kind)
	}
	if len(optional.Subjects) != 1 || optional.Subjects[0].Name != r.ChildName() {
		t.Errorf("optional binding subjects %+v", optional.Subjects)
	}

	var base rbacv1.RoleBinding
	if err := cl.Get(context.Background(), client.ObjectKey{
		Namespace: "rns", Name: r.ChildName(),
	}, &base); err != nil {
		t.Fatalf("base binding missing: %v", err)
	}
	if base.RoleRef.Name != executorClusterRoleName {
		t.Errorf("base binding references %q", base.RoleRef.Name)
	}
}

func TestCreateRBACIfMissingSkipsTheOptionalBindingWhenUnset(t *testing.T) {
	s := crudScheme(t)
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("add corev1: %v", err)
	}
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns", UID: "uid-1"}}
	cl := fake.NewClientBuilder().WithScheme(s).Build()

	if err := CreateRBACIfMissing(context.Background(), cl, cl, r, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bindingExists(t, cl, "rns", pdbBindingPrefix+r.ChildName()) {
		t.Error("an optional binding was created for a runner that never asked for it")
	}
}

// Collision-proofness is the property that no optional name can equal ANY
// runner's base name. Asserted over the shape rather than one example pair: a
// base name always begins "gitlab-runner-", so a prefix that does not start
// with that string cannot produce one.
func TestOptionalBindingNamesCannotCollideWithABaseName(t *testing.T) {
	const basePrefix = "gitlab-runner-"
	if got := (&v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "x"}}).ChildName(); !strings.HasPrefix(got, basePrefix) {
		t.Fatalf("ChildName() is %q, which invalidates this test's premise", got)
	}
	for _, g := range optionalGrants() {
		if g.namePrefix == "" {
			t.Errorf("grant for %q has an empty name prefix, so its binding collides with the base one", g.clusterRole)
			continue
		}
		if strings.HasPrefix(g.namePrefix, basePrefix) {
			t.Errorf("name prefix %q starts with %q, so it can collide with a base binding name",
				g.namePrefix, basePrefix)
		}
	}
}

// recordingClient captures the order of Create calls. Neither the fake client
// nor envtest enforces the apiserver's escalation check, so ordering is the
// only thing a unit test can assert here.
type recordingClient struct {
	client.Client
	order *[]string
}

func (r recordingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	*r.order = append(*r.order, "create "+fmt.Sprintf("%T/%s", obj, obj.GetName()))
	return r.Client.Create(ctx, obj, opts...)
}

func (r recordingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	*r.order = append(*r.order, "update "+fmt.Sprintf("%T/%s", obj, obj.GetName()))
	return r.Client.Update(ctx, obj, opts...)
}

// A RoleBinding whose roleRef names a missing ClusterRole is rejected by the
// apiserver: it resolves roleRef during the escalation check. On a fresh
// cluster that makes creation order load-bearing, and no fake client will
// catch it, so assert the order directly.
func TestClusterRolesAreCreatedBeforeAnyBinding(t *testing.T) {
	s := crudScheme(t)
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("add corev1: %v", err)
	}
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns", UID: "uid-1"}}
	r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()

	var order []string
	base := fake.NewClientBuilder().WithScheme(s).Build()
	cl := recordingClient{Client: base, order: &order}

	if err := CreateRBACIfMissing(context.Background(), cl, base, r, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	indexOf := func(want string) int {
		for i, got := range order {
			if got == want {
				return i
			}
		}
		return -1
	}
	roles := map[string]int{
		executorClusterRoleName:    indexOf("create *v1.ClusterRole/" + executorClusterRoleName),
		executorPDBClusterRoleName: indexOf("create *v1.ClusterRole/" + executorPDBClusterRoleName),
	}
	bindings := map[string]int{
		r.ChildName():                    indexOf("create *v1.RoleBinding/" + r.ChildName()),
		pdbBindingPrefix + r.ChildName(): indexOf("create *v1.RoleBinding/" + pdbBindingPrefix + r.ChildName()),
	}
	for name, at := range roles {
		if at < 0 {
			t.Fatalf("clusterrole %q was never created (order: %v)", name, order)
		}
	}
	for name, at := range bindings {
		if at < 0 {
			t.Fatalf("rolebinding %q was never created (order: %v)", name, order)
		}
		for role, roleAt := range roles {
			if roleAt > at {
				t.Errorf("rolebinding %q is created before clusterrole %q; the apiserver would reject it", name, role)
			}
		}
	}
}

// The optional binding name is observable contract: it is documented in the
// README and hardcoded in the controller and e2e suites, which cannot import
// this package's unexported constant. Pin it here so a change fails loudly
// rather than silently making those assertions query a name nothing creates.
func TestOptionalBindingNameIsStable(t *testing.T) {
	r := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "rns"}}
	r.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
	want := "pdb-" + r.ChildName()
	var got []string
	for _, k := range DesiredBindings(r, []string{"rns"}) {
		if k.ClusterRole == executorPDBClusterRoleName {
			got = append(got, k.Name)
		}
	}
	if len(got) != 1 || got[0] != want {
		t.Errorf("optional binding named %v, want [%q]; update README and the controller/e2e suites together", got, want)
	}
}

// The upgrade ordering commit 782f336 exists to establish, asserted through the
// production entry point. Seeded like a pre-#64 cluster: the base role still
// carries the PDB rule and no optional role exists.
func TestUpgradeGrantsBeforeItRevokes(t *testing.T) {
	legacy := append(desiredRoleRules(), desiredPDBRules()...)
	var order []string
	base := rbacClient(t, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorClusterRoleName},
		Rules:      legacy,
	})
	cl := recordingClient{Client: base, order: &order}
	r := testRunner(true)

	if err := CreateRBACIfMissing(context.Background(), cl, base, r, logr.Discard()); err != nil {
		t.Fatalf("CreateRBACIfMissing: %v", err)
	}

	idx := func(want string) int {
		for i, got := range order {
			if got == want {
				return i
			}
		}
		return -1
	}
	revoke := idx("update *v1.ClusterRole/" + executorClusterRoleName)
	grant := idx("create *v1.RoleBinding/" + pdbBindingPrefix + r.ChildName())
	if revoke < 0 {
		t.Fatalf("the base role was never converged, so the unconditional grant survives (order: %v)", order)
	}
	if grant < 0 {
		t.Fatalf("the optional binding was never created (order: %v)", order)
	}
	if revoke < grant {
		t.Errorf("the base role is stripped at %d before the optional binding is created at %d; "+
			"this runner loses the permission mid-reconcile (order: %v)", revoke, grant, order)
	}
	// And the revocation actually landed.
	var got rbacv1.ClusterRole
	if err := base.Get(context.Background(),
		client.ObjectKey{Name: executorClusterRoleName}, &got); err != nil {
		t.Fatalf("get base clusterrole: %v", err)
	}
	if pdbRule(got.Rules) != nil {
		t.Error("the unconditional poddisruptionbudgets grant survived the reconcile")
	}
}

// The prune must run from the reconcile, not just be correct in isolation.
func TestCreateRBACIfMissingRevokesAStaleOptionalBinding(t *testing.T) {
	r := testRunner(false) // flag now off
	stale := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: pdbBindingPrefix + r.ChildName(), Namespace: "rns", Labels: rbacLabels(r),
		},
	}
	cl := rbacClient(t, stale)
	reconcileRBAC(t, cl, r)

	if bindingExists(t, cl, "rns", pdbBindingPrefix+r.ChildName()) {
		t.Error("a stale optional binding survived the reconcile; the grant is never revoked")
	}
	if !bindingExists(t, cl, "rns", r.ChildName()) {
		t.Error("the base binding is missing")
	}
}

// Drift on the optional role must converge too. Without this the base role has
// three drift tests and the new role has none, so a later rule correction would
// reach fresh installs only.
func TestOptionalClusterRoleConvergesOnDrift(t *testing.T) {
	cl := rbacClient(t, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorPDBClusterRoleName},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{"policy"}, Resources: []string{"poddisruptionbudgets"}, Verbs: []string{"get"}},
		},
	})
	reconcileRBAC(t, cl, testRunner(true))

	var got rbacv1.ClusterRole
	if err := cl.Get(context.Background(),
		client.ObjectKey{Name: executorPDBClusterRoleName}, &got); err != nil {
		t.Fatalf("get optional clusterrole: %v", err)
	}
	rule := pdbRule(got.Rules)
	if rule == nil {
		t.Fatal("no poddisruptionbudgets rule after reconcile")
	}
	if !contains(rule.Verbs, "create") {
		t.Errorf("optional role drift was not corrected: verbs %v, want get and create", rule.Verbs)
	}
}
