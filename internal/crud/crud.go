package crud

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	internalTypes "gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func CreateRBACIfMissing(ctx context.Context, cl client.Client, runnerObject internalTypes.RunnerInfo, log logr.Logger) error {
	if err := CreateSaIfMissing(ctx, cl, runnerObject, log); err != nil {
		return err
	}
	if err := CreateRoleIfMissing(ctx, cl, runnerObject, log); err != nil {
		return err
	}
	if err := CreateRoleBindingIfMissing(ctx, cl, runnerObject, log); err != nil {
		return err
	}
	return nil
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

func CreateRoleIfMissing(ctx context.Context, cl client.Client, runnerObject internalTypes.RunnerInfo, log logr.Logger) error {
	namespacedKey := client.ObjectKey{Namespace: runnerObject.GetNamespace(), Name: runnerObject.ChildName()}
	err := cl.Get(ctx, namespacedKey, &v1.Role{})
	switch {
	case err == nil:
		return nil
	case !errors.IsNotFound(err):
		log.Error(err, "cannot get the role")
		return err
	}
	// sa doesn't exists, create it
	log.Info("creating missing role")
	role := &v1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            runnerObject.ChildName(),
			Namespace:       runnerObject.GetNamespace(),
			OwnerReferences: runnerObject.GenerateOwnerReference(),
		},
		// Minimum permissions the gitlab-runner kubernetes executor needs in the
		// build namespace. Scoped to the core API group, with verbs per resource;
		// no wildcard apiGroup and no namespace-wide secret list/watch.
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/exec", "pods/attach"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/log"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets", "configmaps"},
				Verbs:     []string{"get", "create", "delete"},
			},
		},
	}
	err = cl.Create(ctx, role)
	if err != nil {
		log.Error(err, "cannot create role")
		return err
	}
	return nil
}

func CreateRoleBindingIfMissing(ctx context.Context, cl client.Client, runnerObject internalTypes.RunnerInfo, log logr.Logger) error {
	namespacedKey := client.ObjectKey{Namespace: runnerObject.GetNamespace(), Name: runnerObject.ChildName()}
	err := cl.Get(ctx, namespacedKey, &v1.RoleBinding{})
	switch {
	case err == nil: // service account exists
		return nil
	case !errors.IsNotFound(err):
		log.Error(err, "cannot get the Role binding")
		return err
	}
	// sa doesn't exists, create it
	log.Info("creating missing rolebinding")
	role := &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            runnerObject.ChildName(),
			Namespace:       runnerObject.GetNamespace(),
			OwnerReferences: runnerObject.GenerateOwnerReference(),
		},
		Subjects: []v1.Subject{{
			Kind:      "ServiceAccount",
			Name:      runnerObject.ChildName(),
			Namespace: runnerObject.GetNamespace(),
		}},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     runnerObject.ChildName(),
		},
	}
	err = cl.Create(ctx, role)
	if err != nil {
		log.Error(err, "cannot create rolebindings")
		return err
	}
	return nil
}
