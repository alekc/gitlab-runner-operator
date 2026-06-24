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

// CreateRBACIfMissing reconciles the runner's ServiceAccount, Role, and
// RoleBinding. The SA lives in the runner's namespace (the manager Deployment
// mounts it). A Role + RoleBinding is reconciled in every namespace the
// executor entries target, so the single SA has permissions wherever job pods
// run. RBAC in namespaces no longer targeted is pruned.
func CreateRBACIfMissing(ctx context.Context, cl client.Client, runnerObject internalTypes.RunnerInfo, log logr.Logger) error {
	if err := CreateSaIfMissing(ctx, cl, runnerObject, log); err != nil {
		return err
	}
	namespaces := BuildNamespaces(runnerObject)
	for _, namespace := range namespaces {
		if err := reconcileRole(ctx, cl, runnerObject, namespace, log); err != nil {
			return err
		}
		if err := reconcileRoleBinding(ctx, cl, runnerObject, namespace, log); err != nil {
			return err
		}
	}
	return DeleteRBACExcept(ctx, cl, runnerObject, namespaces, log)
}

// BuildNamespaces returns the distinct namespaces the object's executor entries
// target (executor_config.namespace, defaulting to the object's namespace).
func BuildNamespaces(obj internalTypes.RunnerInfo) []string {
	set := map[string]struct{}{}
	for _, cfg := range obj.ExecutorConfigs() {
		namespace := obj.GetNamespace()
		if cfg != nil && cfg.Namespace != "" {
			namespace = cfg.Namespace
		}
		set[namespace] = struct{}{}
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
// needs in a build namespace. Source of truth: the kubernetes executor RBAC
// reference in the GitLab Runner docs. Optional, feature-flag-gated resources
// (namespaces, poddisruptionbudgets, autoscaler) are intentionally omitted:
// namespace_per_job is rejected by the webhook, and the rest are not modelled.
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

// reconcileRole creates the runner Role in namespace if missing, or updates its
// rules to the desired set if they drifted (so rule changes reach existing
// runners, not only new ones). Owner references are set only when the Role is
// in the runner's own namespace; cross-namespace owner refs are invalid and
// would be garbage collected, so build-namespace RBAC is cleaned up explicitly.
func reconcileRole(ctx context.Context, cl client.Client, obj internalTypes.RunnerInfo, namespace string, log logr.Logger) error {
	desired := desiredRoleRules()
	existing := &v1.Role{}
	key := client.ObjectKey{Namespace: namespace, Name: obj.ChildName()}
	err := cl.Get(ctx, key, existing)
	switch {
	case errors.IsNotFound(err):
		role := &v1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      obj.ChildName(),
				Namespace: namespace,
				Labels:    rbacLabels(obj),
			},
			Rules: desired,
		}
		if namespace == obj.GetNamespace() {
			role.OwnerReferences = obj.GenerateOwnerReference()
		}
		log.Info("creating runner role", "namespace", namespace)
		return cl.Create(ctx, role)
	case err != nil:
		log.Error(err, "cannot get the role", "namespace", namespace)
		return err
	}
	if !equality.Semantic.DeepEqual(existing.Rules, desired) {
		existing.Rules = desired
		log.Info("updating runner role rules", "namespace", namespace)
		return cl.Update(ctx, existing)
	}
	return nil
}

// reconcileRoleBinding creates the runner RoleBinding in namespace if missing.
// The subject is the runner SA (always in the runner's namespace); the roleRef
// is immutable and the subject is stable, so an existing binding needs no
// update.
func reconcileRoleBinding(ctx context.Context, cl client.Client, obj internalTypes.RunnerInfo, namespace string, log logr.Logger) error {
	existing := &v1.RoleBinding{}
	key := client.ObjectKey{Namespace: namespace, Name: obj.ChildName()}
	err := cl.Get(ctx, key, existing)
	switch {
	case errors.IsNotFound(err):
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
				Kind:     "Role",
				Name:     obj.ChildName(),
			},
		}
		if namespace == obj.GetNamespace() {
			binding.OwnerReferences = obj.GenerateOwnerReference()
		}
		log.Info("creating runner rolebinding", "namespace", namespace)
		return cl.Create(ctx, binding)
	case err != nil:
		log.Error(err, "cannot get the rolebinding", "namespace", namespace)
		return err
	}
	return nil
}

// DeleteRBACExcept deletes the operator-managed Role/RoleBinding objects for the
// runner that live in a namespace not present in keep. With keep set to the
// current build namespaces it prunes RBAC left behind when an executor
// namespace is removed from the spec. With keep set to just the runner's own
// namespace it is the finalizer's cross-namespace cleanup (same-namespace RBAC
// carries owner references and is garbage collected by Kubernetes).
func DeleteRBACExcept(ctx context.Context, cl client.Client, obj internalTypes.RunnerInfo, keep []string, log logr.Logger) error {
	keepSet := map[string]struct{}{}
	for _, namespace := range keep {
		keepSet[namespace] = struct{}{}
	}
	selector := client.MatchingLabels(rbacLabels(obj))

	var roles v1.RoleList
	if err := cl.List(ctx, &roles, selector); err != nil {
		return err
	}
	for i := range roles.Items {
		if _, ok := keepSet[roles.Items[i].Namespace]; ok {
			continue
		}
		log.Info("pruning runner role", "namespace", roles.Items[i].Namespace)
		if err := cl.Delete(ctx, &roles.Items[i]); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	var bindings v1.RoleBindingList
	if err := cl.List(ctx, &bindings, selector); err != nil {
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
