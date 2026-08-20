package crud

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	internalTypes "gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RBAC object labels. The owner labels scope a List to a single runner object
// so RBAC can be pruned (on namespace change) and cleaned up across namespaces
// (on deletion) without relying on owner references, which are invalid across
// namespaces.
const (
	managedByLabel      = "app.kubernetes.io/managed-by"
	managedByValue      = "gitlab-runner-operator"
	ownerNamespaceLabel = "gitlab.k8s.alekc.dev/owner-namespace"
	ownerUIDLabel       = "gitlab.k8s.alekc.dev/owner-uid"

	// executorClusterRoleName is the shared ClusterRole holding the permissions
	// every kubernetes executor needs. Every runner RoleBinding references it,
	// so the rules are defined once instead of duplicated per runner.
	executorClusterRoleName = "gitlab-runner-operator-executor"

	// executorPDBClusterRoleName carries the pod_disruption_budget grant alone,
	// bound only in build namespaces whose executor entries enable the flag.
	// One role per optional permission, so opting into a future one does not
	// hand out this one as well.
	executorPDBClusterRoleName = "gitlab-runner-operator-executor-pdb"

	// pdbBindingPrefix separates the optional binding from the base one. A
	// prefix, not a suffix: ChildName() always starts "gitlab-runner-", so a
	// prefixed name can never equal another runner's base binding, whereas
	// "<child>-pdb" collides with a runner literally named "<name>-pdb".
	pdbBindingPrefix = "pdb-"
)

// SingleRunner init single runner from k8s
func SingleRunner(ctx context.Context, client client.Client, nsName types.NamespacedName) (internalTypes.RunnerInfo, error) {
	runnerObj := &gitlabv1beta2.Runner{}
	err := client.Get(ctx, nsName, runnerObj)
	return runnerObj, err
}

// MultiRunner fetches multirunner from k8s
// todo: generics?
func MultiRunner(ctx context.Context, client client.Client, nsName types.NamespacedName) (internalTypes.RunnerInfo, error) {
	runnerObj := &gitlabv1beta2.MultiRunner{}
	err := client.Get(ctx, nsName, runnerObj)

	if runnerObj.Status.RunnerIDs == nil {
		runnerObj.Status.RunnerIDs = map[string]int{}
	}
	if runnerObj.Status.RegistrationHashes == nil {
		runnerObj.Status.RegistrationHashes = map[string]string{}
	}
	if runnerObj.Status.TokenExpiresAt == nil {
		runnerObj.Status.TokenExpiresAt = map[string]metav1.Time{}
	}

	return runnerObj, err
}

// ExistingConfigTokens recovers the per-entry authentication tokens stored in
// the runner's config Secret (keys prefixed with ConfigTokenKeyPrefix). It
// returns an empty map when the Secret does not exist yet. The token lives only
// in this Secret, never in the CR status.
func ExistingConfigTokens(ctx context.Context, cl client.Client, namespace, childName string) (map[string]string, error) {
	out := map[string]string{}
	var secret corev1.Secret
	err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: childName}, &secret)
	if errors.IsNotFound(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for k, v := range secret.Data {
		if name, ok := strings.CutPrefix(k, internalTypes.ConfigTokenKeyPrefix); ok {
			out[name] = string(v)
		}
	}
	return out, nil
}

// ExistingConfigCA returns the custom CA bundle persisted in the runner's config
// Secret (the CACertFileName key), or nil when the Secret or key is absent. The
// delete path uses it so unregistration does not depend on the user's CA
// Secret/ConfigMap, which may already be gone at finalization.
func ExistingConfigCA(ctx context.Context, cl client.Client, namespace, childName string) ([]byte, error) {
	var secret corev1.Secret
	err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: childName}, &secret)
	if errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return secret.Data[internalTypes.CACertFileName], nil
}

// CreateRBACIfMissing reconciles the runner's RBAC. The permission set lives in
// one shared ClusterRole; each runner gets its own ServiceAccount (distinct
// identity, audit, and lifecycle) and a RoleBinding in every namespace its
// executor entries target, binding that SA to the shared ClusterRole. The SA
// stays in the runner's namespace (the manager Deployment mounts it).
// RoleBindings in namespaces no longer targeted are pruned.
func CreateRBACIfMissing(ctx context.Context, cl client.Client, apiReader client.Reader, runnerObject internalTypes.RunnerInfo, log logr.Logger) error {
	// Grant before revoke. The optional roles and this runner's bindings are
	// written first, so converging the base role (which drops a rule that used
	// to be unconditional) never leaves this runner without the permission it
	// still wants. See the upgrade note in ensureBaseClusterRole.
	if err := ensureOptionalClusterRoles(ctx, apiReader, cl, log); err != nil {
		return err
	}
	if err := CreateSaIfMissing(ctx, cl, runnerObject, log); err != nil {
		return err
	}
	desired := DesiredBindings(runnerObject, BuildNamespaces(runnerObject))
	for _, key := range desired {
		if err := reconcileRoleBinding(ctx, cl, runnerObject, key.Namespace, key.Name, key.ClusterRole, log); err != nil {
			return err
		}
	}
	if err := ensureBaseClusterRole(ctx, apiReader, cl, log); err != nil {
		return err
	}
	return DeleteRBACExcept(ctx, cl, apiReader, runnerObject, desired, log)
}

// optionalGrant is an executor permission gated behind a spec field. Each has
// its own ClusterRole, so opting into one does not hand out the others.
type optionalGrant struct {
	namePrefix  string
	clusterRole string
	rules       []v1.PolicyRule
	// wanted reports the build namespaces whose entries enable this grant.
	wanted func(obj internalTypes.RunnerInfo) map[string]struct{}
}

func optionalGrants() []optionalGrant {
	return []optionalGrant{{
		namePrefix:  pdbBindingPrefix,
		clusterRole: executorPDBClusterRoleName,
		rules:       desiredPDBRules(),
		wanted:      pdbNamespaces,
	}}
}

// BindingKey identifies one operator-managed RoleBinding and the ClusterRole it
// grants. Callers pass these to DeleteRBACExcept, so the prune compares against
// the bindings that should exist rather than against namespaces alone, which
// cannot see an optional grant that a spec change has just turned off.
type BindingKey struct {
	Namespace   string
	Name        string
	ClusterRole string
}

// DesiredBindings returns the RoleBindings the object should have across the
// given namespaces: the base binding everywhere, plus the optional
// pod_disruption_budget binding in namespaces whose entries enable it.
func DesiredBindings(obj internalTypes.RunnerInfo, namespaces []string) []BindingKey {
	grants := optionalGrants()
	wanted := make([]map[string]struct{}, len(grants))
	for i, g := range grants {
		wanted[i] = g.wanted(obj)
	}
	out := make([]BindingKey, 0, len(namespaces))
	for _, namespace := range namespaces {
		out = append(out, BindingKey{
			Namespace:   namespace,
			Name:        obj.ChildName(),
			ClusterRole: executorClusterRoleName,
		})
		for i, g := range grants {
			if _, ok := wanted[i][namespace]; !ok {
				continue
			}
			out = append(out, BindingKey{
				Namespace:   namespace,
				Name:        g.namePrefix + obj.ChildName(),
				ClusterRole: g.clusterRole,
			})
		}
	}
	return out
}

// AllBindingsIn returns every binding name the object can own in a namespace,
// wanted or not. The finalizer keeps same-namespace bindings (owner references
// collect them) regardless of whether the optional grant is currently on.
func AllBindingsIn(obj internalTypes.RunnerInfo, namespace string) []BindingKey {
	out := []BindingKey{
		{Namespace: namespace, Name: obj.ChildName(), ClusterRole: executorClusterRoleName},
	}
	for _, g := range optionalGrants() {
		out = append(out, BindingKey{
			Namespace:   namespace,
			Name:        g.namePrefix + obj.ChildName(),
			ClusterRole: g.clusterRole,
		})
	}
	return out
}

// ensureBaseClusterRole converges the shared role every executor needs. It is
// shared, so the first runner to reconcile after an upgrade drops a removed
// rule for the whole fleet, and each runner that still wants it regains it only
// on its own next reconcile. Reconciled last for that reason.
func ensureBaseClusterRole(ctx context.Context, apiReader client.Reader, cl client.Client, log logr.Logger) error {
	return ensureClusterRole(ctx, apiReader, cl, executorClusterRoleName, desiredRoleRules(), log)
}

// ensureOptionalClusterRoles converges one ClusterRole per optional grant. They
// are created whether or not anything binds them: an unbound ClusterRole grants
// nothing, and having it present means a binding never races its role.
func ensureOptionalClusterRoles(ctx context.Context, apiReader client.Reader, cl client.Client, log logr.Logger) error {
	for _, g := range optionalGrants() {
		if err := ensureClusterRole(ctx, apiReader, cl, g.clusterRole, g.rules, log); err != nil {
			return err
		}
	}
	return nil
}

// ensureExecutorClusterRoles converges every shared executor ClusterRole. Used
// where ordering does not matter; CreateRBACIfMissing splits the two halves so
// it can grant before it revokes.
func ensureExecutorClusterRoles(ctx context.Context, apiReader client.Reader, cl client.Client, log logr.Logger) error {
	if err := ensureOptionalClusterRoles(ctx, apiReader, cl, log); err != nil {
		return err
	}
	return ensureBaseClusterRole(ctx, apiReader, cl, log)
}

// ensureClusterRole converges one shared ClusterRole on its desired rules. It
// carries no owner reference (it outlives any one runner). The operator holds
// these permissions itself, so the apiserver escalation check permits writing
// the role and binding runner SAs to it.
//
// The read uses the uncached APIReader to avoid a cluster-wide informer on a
// type the operator neither owns nor watches; writes use the cached client.
func ensureClusterRole(
	ctx context.Context,
	apiReader client.Reader,
	cl client.Client,
	name string,
	desired []v1.PolicyRule,
	log logr.Logger,
) error {
	existing := &v1.ClusterRole{}
	err := apiReader.Get(ctx, client.ObjectKey{Name: name}, existing)
	switch {
	case errors.IsNotFound(err):
		role := &v1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{managedByLabel: managedByValue},
			},
			Rules: desired,
		}
		log.Info("creating shared executor clusterrole", "name", name)
		// The ClusterRole is shared, so many runner reconciles race to create
		// it; the loser seeing AlreadyExists has nothing to do (whoever won
		// wrote the same desired rules).
		if err := cl.Create(ctx, role); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
		return nil
	case err != nil:
		log.Error(err, "cannot get the executor clusterrole")
		return err
	}
	if !equality.Semantic.DeepEqual(existing.Rules, desired) {
		existing.Rules = desired
		log.Info("updating shared executor clusterrole rules", "name", name)
		// Concurrent reconciles all compute identical desired rules, so a lost
		// optimistic-lock race means another reconcile already converged it.
		if err := cl.Update(ctx, existing); err != nil && !errors.IsConflict(err) {
			return err
		}
	}
	return nil
}

// BuildNamespaces returns the distinct namespaces the object's executor entries
// target (executor_config.namespace, defaulting to the object's namespace).
func BuildNamespaces(obj internalTypes.RunnerInfo) []string {
	set := map[string]struct{}{}
	for _, cfg := range obj.ExecutorConfigs() {
		// EffectiveNamespace is the shared defaulting rule used by config.toml
		// rendering too, so RBAC is always provisioned for the namespace jobs
		// actually run in.
		set[cfg.EffectiveNamespace(obj.GetNamespace())] = struct{}{}
	}
	if len(set) == 0 {
		set[obj.GetNamespace()] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for namespace := range set {
		out = append(out, namespace)
	}
	return out
}

// desiredRoleRules is the permission set the kubernetes executor always needs,
// applied to the shared executor ClusterRole. Source of truth: the executor
// RBAC reference in the GitLab Runner docs. namespaces and deployments are
// omitted (namespace_per_job is CEL-rejected, the autoscaler is not exposed).
func desiredRoleRules() []v1.PolicyRule {
	return []v1.PolicyRule{
		{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list", "watch", "create", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"pods/exec", "pods/attach"}, Verbs: []string{"get", "create", "patch", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"pods/log"}, Verbs: []string{"get", "list"}},
		{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get", "create", "update", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get", "create", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"services"}, Verbs: []string{"get", "create"}},
		{APIGroups: []string{""}, Resources: []string{"serviceaccounts"}, Verbs: []string{"get"}},
		{APIGroups: []string{""}, Resources: []string{"events"}, Verbs: []string{"list", "watch"}},
	}
}

// desiredPDBRules is the grant pod_disruption_budget needs. It is separate
// because upstream defaults the flag to false, so a runner that never sets it
// creates no PDB; when it is set, a missing verb errors the whole job.
func desiredPDBRules() []v1.PolicyRule {
	return []v1.PolicyRule{
		// The executor only creates and reads the PDB; the build pod owns it, so
		// deletion happens by garbage collection rather than an explicit call.
		{APIGroups: []string{"policy"}, Resources: []string{"poddisruptionbudgets"}, Verbs: []string{"get", "create"}},
	}
}

// pdbNamespaces reports the build namespaces whose executor entries enable
// pod_disruption_budget. Per namespace rather than per object: the PDB is
// created next to the build pod, so a MultiRunner enabling it on one entry
// must not widen the grant in namespaces its other entries target.
func pdbNamespaces(obj internalTypes.RunnerInfo) map[string]struct{} {
	out := map[string]struct{}{}
	for _, cfg := range obj.ExecutorConfigs() {
		if cfg.PodDisruptionBudget != nil && *cfg.PodDisruptionBudget {
			out[cfg.EffectiveNamespace(obj.GetNamespace())] = struct{}{}
		}
	}
	return out
}

// rbacLabels scope a List to one runner object's RBAC across namespaces. The
// owner UID is used rather than the name because it is always a valid label
// value (a long object name would exceed the 63-character limit) and is unique.
func rbacLabels(obj internalTypes.RunnerInfo) map[string]string {
	return map[string]string{
		managedByLabel:      managedByValue,
		ownerNamespaceLabel: obj.GetNamespace(),
		ownerUIDLabel:       string(obj.GetUID()),
	}
}

func CreateSaIfMissing(ctx context.Context, cl client.Client, runnerObject internalTypes.RunnerInfo, log logr.Logger) error {
	namespacedKey := client.ObjectKey{Namespace: runnerObject.GetNamespace(), Name: runnerObject.ChildName()}
	err := cl.Get(ctx, namespacedKey, &corev1.ServiceAccount{})
	switch {
	case err == nil: // service account exists
		return nil
	case !errors.IsNotFound(err):
		log.Error(err, "cannot get the service account")
		return err
	}
	// sa doesn't exists, create it
	log.Info("creating missing sa")
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:            runnerObject.ChildName(),
			Namespace:       runnerObject.GetNamespace(),
			OwnerReferences: runnerObject.GenerateOwnerReference(),
		},
	}
	if err = cl.Create(ctx, sa); err != nil {
		log.Error(err, "cannot create service-account")
		return err
	}
	return nil
}

// desiredRoleBinding builds the RoleBinding that binds the runner's SA to the
// shared executor ClusterRole in namespace. Owner references are set only in
// the runner's own namespace; cross-namespace owner refs are invalid and would
// be garbage collected, so build-namespace bindings are cleaned up explicitly.
func desiredRoleBinding(obj internalTypes.RunnerInfo, namespace, name, clusterRole string) *v1.RoleBinding {
	binding := &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    rbacLabels(obj),
		},
		Subjects: []v1.Subject{{
			Kind:      "ServiceAccount",
			Name:      obj.ChildName(),
			Namespace: obj.GetNamespace(),
		}},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
	}
	if namespace == obj.GetNamespace() {
		binding.OwnerReferences = obj.GenerateOwnerReference()
	}
	return binding
}

// reconcileRoleBinding creates the runner RoleBinding in namespace if missing.
// The subject SA and the ClusterRole roleRef are stable, so an existing binding
// usually needs no update; if its roleRef differs (for example a binding left
// by an older operator version that referenced a per-runner Role) it is
// recreated, because roleRef is immutable.
func reconcileRoleBinding(
	ctx context.Context,
	cl client.Client,
	obj internalTypes.RunnerInfo,
	namespace, name, clusterRole string,
	log logr.Logger,
) error {
	desired := desiredRoleBinding(obj, namespace, name, clusterRole)
	existing := &v1.RoleBinding{}
	key := client.ObjectKey{Namespace: namespace, Name: name}
	err := cl.Get(ctx, key, existing)
	switch {
	case errors.IsNotFound(err):
		log.Info("creating runner rolebinding", "namespace", namespace)
		// A concurrent reconcile may have created it first; that is success.
		if err := cl.Create(ctx, desired); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
		return nil
	case err != nil:
		log.Error(err, "cannot get the rolebinding", "namespace", namespace)
		return err
	}
	if existing.RoleRef != desired.RoleRef {
		// roleRef is immutable, so a changed binding (e.g. one left by an older
		// operator version referencing a per-runner Role) must be replaced. The
		// gap between delete and create is covered by requeue on error.
		log.Info("recreating runner rolebinding with new roleRef", "namespace", namespace)
		if err := cl.Delete(ctx, existing); err != nil && !errors.IsNotFound(err) {
			return err
		}
		if err := cl.Create(ctx, desired); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

// DeleteRBACExcept deletes the operator-managed RoleBindings for the runner that
// are not in keep, matched on namespace and name. Keying on the binding rather
// than the namespace is what lets an optional grant be revoked: turning
// pod_disruption_budget off leaves the namespace in use, only the binding goes.
//
// With keep from DesiredBindings it also prunes bindings left behind when an
// executor namespace leaves the spec. With keep from AllBindingsIn(own) it is
// the finalizer's cross-namespace cleanup; same-namespace bindings carry owner
// references and are collected by Kubernetes. ClusterRoles and the
// ServiceAccount are not touched here.
//
// The List uses the uncached reader (APIReader): a cross-namespace binding has
// no owner reference, so the cache-list could miss one just created (or, during
// finalization, drop the finalizer before observing it) and orphan a live grant.
func DeleteRBACExcept(ctx context.Context, cl client.Client, reader client.Reader, obj internalTypes.RunnerInfo, keep []BindingKey, log logr.Logger) error {
	keepSet := map[string]struct{}{}
	for _, k := range keep {
		keepSet[k.Namespace+"/"+k.Name] = struct{}{}
	}
	selector := client.MatchingLabels(rbacLabels(obj))

	var bindings v1.RoleBindingList
	if err := reader.List(ctx, &bindings, selector); err != nil {
		return err
	}
	for i := range bindings.Items {
		item := &bindings.Items[i]
		if _, ok := keepSet[item.Namespace+"/"+item.Name]; ok {
			continue
		}
		log.Info("pruning runner rolebinding", "namespace", item.Namespace, "name", item.Name)
		if err := cl.Delete(ctx, item); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
