package crud

import (
	"context"
	"os"
	"path/filepath"
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
	s := crudScheme(t)
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	if err := ensureExecutorClusterRoles(context.Background(), cl, cl, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

	if err := ensureExecutorClusterRoles(context.Background(), cl, cl, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

// The #64 upgrade path: an existing cluster carries poddisruptionbudgets in the
// shared role. Revocation only reaches it through the drift branch, so without
// this every pre-existing install would keep the unconditional grant forever.
func TestEnsureExecutorClusterRoleDropsPDBFromTheBaseRole(t *testing.T) {
	s := crudScheme(t)
	legacy := append(desiredRoleRules(), desiredPDBRules()...)
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorClusterRoleName},
		Rules:      legacy,
	}).Build()

	if err := ensureExecutorClusterRoles(context.Background(), cl, cl, logr.Discard()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	s := crudScheme(t)
	extra := append(desiredRoleRules(), rbacv1.PolicyRule{
		APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"create"},
	})
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: executorClusterRoleName},
		Rules:      extra,
	}).Build()

	if err := ensureExecutorClusterRoles(context.Background(), cl, cl, logr.Discard()); err != nil {
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

// A prefix is collision-proof by construction: ChildName() always begins
// "gitlab-runner-", so no optional name can equal another runner's base name.
func TestOptionalBindingNamesCannotCollideWithABaseName(t *testing.T) {
	victim := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "x-pdb", Namespace: "rns"}}
	attacker := &v1beta2.Runner{ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: "rns"}}
	attacker.Spec.ExecutorConfig.PodDisruptionBudget = pdbTrue()
	for _, g := range optionalGrants() {
		if got := g.namePrefix + attacker.ChildName(); got == victim.ChildName() {
			t.Errorf("optional name %q collides with the base binding of runner %q", got, victim.GetName())
		}
	}
}
