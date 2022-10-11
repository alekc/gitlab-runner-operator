package crud

import (
	"context"

	"github.com/go-logr/logr"
	gitlabv1beta1 "gitlab.k8s.alekc.dev/api/v1beta1"
	internalTypes "gitlab.k8s.alekc.dev/internal/types"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SingleRunner fetches single runner from k8s
func SingleRunner(ctx context.Context, client client.Client, nsName types.NamespacedName) (internalTypes.RunnerInfo, error) {
	runnerObj := &gitlabv1beta1.Runner{}
	err := client.Get(ctx, nsName, runnerObj)
	return runnerObj, err
}

// MultiRunner fetches multirunner from k8s
// todo: generics?
func MultiRunner(ctx context.Context, client client.Client, nsName types.NamespacedName) (internalTypes.RunnerInfo, error) {
	runnerObj := &gitlabv1beta1.MultiRunner{}
	err := client.Get(ctx, nsName, runnerObj)
	return runnerObj, err
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
		Rules: []v1.PolicyRule{{
			Verbs:     []string{"get", "list", "watch", "create", "patch", "delete", "update"},
			APIGroups: []string{"*"},
			Resources: []string{"pods", "pods/exec", "pods/attach", "secrets", "configmaps"},
		}},
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
