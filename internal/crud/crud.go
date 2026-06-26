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

	// executorClusterRoleName is the single, shared ClusterRole that holds the
	// kubernetes executor permission set. Every runner RoleBinding references
	// it, so the rules are defined once instead of duplicated per runner.
	executorClusterRoleName = "gitlab-runner-operator-executor"
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
	if err := ensureExecutorClusterRole(ctx, apiReader, cl, log); err != nil {
		return err
	}
	if err := CreateSaIfMissing(ctx, cl, runnerObject, log); err != nil {
		return err
	}
	namespaces := BuildNamespaces(runnerObject)
	for _, namespace := range namespaces {
		if err := reconcileRoleBinding(ctx, cl, runnerObject, namespace, log); err != nil {
			return err
		}
	}
	return DeleteRBACExcept(ctx, cl, apiReader, runnerObject, namespaces, log)
}

// ensureExecutorClusterRole reconciles the single shared ClusterRole to the
// desired rules. It is not owner-referenced (it outlives any one runner) and is
// created once, then updated only when the rules drift, so a rule change in the
// operator reaches every runner at once. The operator holds these permissions
// itself (manager ClusterRole), so the API server's escalation check permits
// both writing this ClusterRole and binding runner SAs to it. The read uses the
// uncached APIReader so the operator does not spin up a cluster-wide informer on
// every ClusterRole (a type it neither owns nor watches); writes go through the
// regular cached client.
func ensureExecutorClusterRole(ctx context.Context, apiReader client.Reader, cl client.Client, log logr.Logger) error {
	desired := desiredRoleRules()
	existing := &v1.ClusterRole{}
	err := apiReader.Get(ctx, client.ObjectKey{Name: executorClusterRoleName}, existing)
	switch {
	case errors.IsNotFound(err):
		role := &v1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   executorClusterRoleName,
				Labels: map[string]string{managedByLabel: managedByValue},
			},
			Rules: desired,
		}
		log.Info("creating shared executor clusterrole", "name", executorClusterRoleName)
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
		log.Info("updating shared executor clusterrole rules", "name", executorClusterRoleName)
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

// desiredRoleRules is the permission set the gitlab-runner kubernetes executor
// needs, applied to the shared executor ClusterRole. Source of truth: the
// kubernetes executor RBAC reference in the GitLab Runner docs. Optional,
// feature-flag-gated resources (namespaces, poddisruptionbudgets, autoscaler)
// are intentionally omitted: namespace_per_job is rejected by the webhook, and
// the rest are not modelled.
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
func desiredRoleBinding(obj internalTypes.RunnerInfo, namespace string) *v1.RoleBinding {
	binding := &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.ChildName(),
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
			Name:     executorClusterRoleName,
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
func reconcileRoleBinding(ctx context.Context, cl client.Client, obj internalTypes.RunnerInfo, namespace string, log logr.Logger) error {
	desired := desiredRoleBinding(obj, namespace)
	existing := &v1.RoleBinding{}
	key := client.ObjectKey{Namespace: namespace, Name: obj.ChildName()}
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
// live in a namespace not present in keep. With keep set to the current build
// namespaces it prunes bindings left behind when an executor namespace is
// removed from the spec. With keep set to just the runner's own namespace it is
// the finalizer's cross-namespace cleanup (same-namespace bindings carry owner
// references and are garbage collected by Kubernetes). The shared ClusterRole
// and the ServiceAccount are not touched here.
//
// The List uses the uncached reader (APIReader): a cross-namespace binding has
// no owner reference, so the cache-list could miss one just created (or, during
// finalization, drop the finalizer before observing it) and orphan a live grant.
func DeleteRBACExcept(ctx context.Context, cl client.Client, reader client.Reader, obj internalTypes.RunnerInfo, keep []string, log logr.Logger) error {
	keepSet := map[string]struct{}{}
	for _, namespace := range keep {
		keepSet[namespace] = struct{}{}
	}
	selector := client.MatchingLabels(rbacLabels(obj))

	var bindings v1.RoleBindingList
	if err := reader.List(ctx, &bindings, selector); err != nil {
		return err
	}
	for i := range bindings.Items {
		if _, ok := keepSet[bindings.Items[i].Namespace]; ok {
			continue
		}
		log.Info("pruning runner rolebinding", "namespace", bindings.Items[i].Namespace)
		if err := cl.Delete(ctx, &bindings.Items[i]); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
