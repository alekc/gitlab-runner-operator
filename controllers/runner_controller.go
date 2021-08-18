/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	coreErrors "errors"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/internal/api"
	internalErrors "gitlab.k8s.alekc.dev/internal/errors"
	"gitlab.k8s.alekc.dev/internal/generate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gitlabv1beta1 "gitlab.k8s.alekc.dev/api/v1beta1"
)

const ownerCmKey = ".metadata.cmcontroller"
const ownerDpKey = ".metadata.dpcontroller"

const defaultTimeout = 15 * time.Second
const configMapKeyName = "config.toml"
const configVersionAnnotationKey = "config-version"
const lastRegistrationTokenAnnotationKey = "last-reg-token"
const lastRegistrationTags = "last-reg-tags"

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	GitlabApiClient api.GitlabClient
}

var resultRequeueAfterDefaultTimeout = ctrl.Result{Requeue: true, RequeueAfter: defaultTimeout}

// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=runners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=runners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=runners/finalizers,verbs=update
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="core",resources=configmaps;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=*

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("starting")

	// find the object, in case we cannot find it just return.
	runnerObj := &gitlabv1beta1.Runner{}
	err := r.Client.Get(ctx, req.NamespacedName, runnerObj)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// reset the error
	runnerObj.Status.Error = ""

	// Sanity check. Initialize our annotations if needed.
	if runnerObj.Annotations == nil {
		runnerObj.Annotations = make(map[string]string, 0)
	}

	// create required rbac credentials if they are missing
	if err = r.CreateRBACIfMissing(ctx, runnerObj, logger); err != nil {
		runnerObj.Status.Error = "Cannot create the rbac objects"
		if err := r.Client.Status().Update(ctx, runnerObj); err != nil {
			logger.Error(err, "cannot patch runner status")
		}
		return ctrl.Result{RequeueAfter: defaultTimeout}, err
	}

	// if the runner doesn't have a saved authentication token
	// or the latest registration token/tags are different from the
	// current one, we need to redo the registration
	if runnerObj.Status.AuthenticationToken == "" ||
		runnerObj.GetAnnotation(lastRegistrationTokenAnnotationKey) != *runnerObj.Spec.RegistrationConfig.Token ||
		runnerObj.GetAnnotation(lastRegistrationTags) != strings.Join(runnerObj.Spec.RegistrationConfig.TagList, ",") {
		// todo: delete old runner
		return r.RegisterNewRunnerOnGitlab(ctx, runnerObj, logger)
	}

	// generate a new config map based on the runner spec
	gitlabRunnerConfig, configHashKey, err := generate.ConfigText(runnerObj)
	if err != nil {
		logger.Error(err, "cannot generate config map")
		return resultRequeueAfterDefaultTimeout, err
	}

	// set the status with a config map hash
	// if the config version differs, perform the upgrade
	if runnerObj.Status.ConfigMapVersion != configHashKey {
		logger.Info("a new version of config map detected. updating Runner",
			"new_version", configHashKey,
			"old_version", runnerObj.Status.ConfigMapVersion)
		runnerObj.Status.ConfigMapVersion = configHashKey
		if err = r.Client.Status().Update(ctx, runnerObj); err != nil {
			logger.Error(err, "cannot assign config map hash to the runner")
			return resultRequeueAfterDefaultTimeout, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	result, err := r.ValidateConfigMap(ctx, runnerObj, gitlabRunnerConfig, configHashKey)
	switch {
	case err != nil:
		return resultRequeueAfterDefaultTimeout, err
	case result != nil:
		return *result, nil
	}

	result, err = r.ValidateDeployment(ctx, runnerObj)
	switch {
	case err != nil:
		return resultRequeueAfterDefaultTimeout, err
	case result != nil:
		return *result, nil
	}

	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) ValidateDeployment(ctx context.Context, runnerObj *gitlabv1beta1.Runner) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)

	labels := map[string]string{"deployment": runnerObj.Name}
	wantedDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runnerObj.ChildName(),
			Namespace: runnerObj.Namespace,
			Annotations: map[string]string{
				configVersionAnnotationKey: runnerObj.Status.ConfigMapVersion,
			},
			OwnerReferences: runnerObj.GenerateOwnerReference(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						configVersionAnnotationKey: runnerObj.Status.ConfigMapVersion,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: runnerObj.ChildName()},
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "runner",
						Image:           "gitlab/gitlab-runner:alpine-v14.0.1",
						Resources:       corev1.ResourceRequirements{}, //todo:
						ImagePullPolicy: "IfNotPresent",                //todo
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/etc/gitlab-runner/",
						}},
					}},
					ServiceAccountName: runnerObj.ChildName(),
				},
			},
		},
	}

	var existingDeployment appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{
		Namespace: runnerObj.Namespace,
		Name:      runnerObj.ChildName(),
	}, &existingDeployment)

	// if deployment doesn't exist, create it
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "could not obtain the deployment")
			return &resultRequeueAfterDefaultTimeout, err
		}

		if err = r.Client.Create(ctx, &wantedDeployment); err != nil {
			logger.Error(err, "cannot create a deployment", "deploymentName", existingDeployment.Name)
			return &resultRequeueAfterDefaultTimeout, err
		}
		return &ctrl.Result{Requeue: true}, nil
	}

	// deployment exists. Check the configMap annotation
	if existingDeployment.GetAnnotations()[configVersionAnnotationKey] != runnerObj.Status.
		ConfigMapVersion || !equality.Semantic.DeepDerivative(wantedDeployment.Spec, existingDeployment.DeepCopy().Spec) {
		logger.Info("deployment is different from our version, updating", "deployment_name", existingDeployment.Name)
		err = r.Client.Update(ctx, &wantedDeployment)
		if err != nil {
			logger.Error(err, "cannot update deployment")
			return &resultRequeueAfterDefaultTimeout, err
		}
	}

	return nil, nil
}

func (r *RunnerReconciler) ValidateConfigMap(ctx context.Context, runnerObj *gitlabv1beta1.Runner,
	gitlabRunnerTomlConfig string,
	hashKey string) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var configMap corev1.ConfigMap
	// fetch all config maps who are registered to the runner as owner
	err := r.Get(ctx, client.ObjectKey{
		Namespace: runnerObj.Namespace,
		Name:      runnerObj.ChildName(),
	}, &configMap)

	// create config map (or report an error)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "could not obtain the config map")
			return &resultRequeueAfterDefaultTimeout, err
		}
		// configmap doesn't exist, create it and requeue
		configMap = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            runnerObj.ChildName(),
				Namespace:       runnerObj.Namespace,
				OwnerReferences: runnerObj.GenerateOwnerReference(),
				Annotations:     map[string]string{configVersionAnnotationKey: hashKey},
			},
			Data: map[string]string{configMapKeyName: gitlabRunnerTomlConfig},
		}
		if err = r.Client.Create(ctx, &configMap); err != nil {
			logger.Error(err, "cannot create a config map", "configMapName", configMap.Name)
			return &resultRequeueAfterDefaultTimeout, err
		}
		return &ctrl.Result{Requeue: true}, nil
	}

	// configmap exists. Check the value for the config map is different
	if configMap.Data[configMapKeyName] != gitlabRunnerTomlConfig || configMap.
		Annotations[configVersionAnnotationKey] != hashKey {
		logger.Info("config map object has old config, needs updating", "configMapName", configMap.Name)
		newObj := configMap.DeepCopy()
		newObj.Data[configMapKeyName] = gitlabRunnerTomlConfig
		newObj.Annotations[configVersionAnnotationKey] = hashKey
		if err = r.Update(ctx, newObj); err != nil && !internalErrors.IsStale(err) {
			logger.Error(
				err,
				"cannot update config map with the new configuration",
				"config_map_name", configMap.Name)
			return &ctrl.Result{Requeue: true}, err
		}
	}

	return nil, nil
}

func (r *RunnerReconciler) createSAIfMissing(ctx context.Context, runnerObject *gitlabv1beta1.Runner, log logr.Logger) error {
	namespacedKey := client.ObjectKey{Namespace: runnerObject.Namespace, Name: runnerObject.ChildName()}
	err := r.Client.Get(ctx, namespacedKey, &corev1.ServiceAccount{})
	switch {
	case err == nil: // service account exists
		return nil
	case !errors.IsNotFound(err):
		log.Error(err, "cannot get the service account")
		return err
	}
	// sa doesn't exists, create it
	log.Info("creating missing s")
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:            runnerObject.ChildName(),
			Namespace:       runnerObject.Namespace,
			OwnerReferences: runnerObject.GenerateOwnerReference(),
		},
	}
	if err = r.Client.Create(ctx, sa); err != nil {
		log.Error(err, "cannot create service-account")
		return err
	}
	return nil
}

func (r *RunnerReconciler) createRoleIfMissing(ctx context.Context, runnerObject *gitlabv1beta1.Runner,
	log logr.Logger) error {
	namespacedKey := client.ObjectKey{Namespace: runnerObject.Namespace, Name: runnerObject.ChildName()}
	err := r.Client.Get(ctx, namespacedKey, &v1.Role{})
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
			Namespace:       runnerObject.Namespace,
			OwnerReferences: runnerObject.GenerateOwnerReference(),
		},
		Rules: []v1.PolicyRule{{
			Verbs:     []string{"get", "list", "watch", "create", "patch", "delete"},
			APIGroups: []string{"*"},
			Resources: []string{"pods", "pods/exec", "pods/attach", "secrets", "configmaps"},
		}},
	}
	err = r.Client.Create(ctx, role)
	if err != nil {
		log.Error(err, "cannot create role")
		return err
	}
	return nil
}

func (r *RunnerReconciler) createRoleBindingIfMissing(ctx context.Context, runnerObject *gitlabv1beta1.Runner,
	log logr.Logger) error {
	namespacedKey := client.ObjectKey{Namespace: runnerObject.Namespace, Name: runnerObject.ChildName()}
	err := r.Client.Get(ctx, namespacedKey, &v1.RoleBinding{})
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
			Namespace:       runnerObject.Namespace,
			OwnerReferences: runnerObject.GenerateOwnerReference(),
		},
		Subjects: []v1.Subject{{
			Kind:      "ServiceAccount",
			Name:      runnerObject.ChildName(),
			Namespace: runnerObject.Namespace,
		}},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     runnerObject.ChildName(),
		},
	}
	err = r.Client.Create(ctx, role)
	if err != nil {
		log.Error(err, "cannot create rolebindings")
		return err
	}
	return nil
}

// CreateRBACIfMissing creates missing rbacs if needed
// todo: introduce the patching?
func (r *RunnerReconciler) CreateRBACIfMissing(ctx context.Context, runnerObject *gitlabv1beta1.Runner, log logr.Logger) error {
	if err := r.createSAIfMissing(ctx, runnerObject, log); err != nil {
		return err
	}
	if err := r.createRoleIfMissing(ctx, runnerObject, log); err != nil {
		return err
	}
	if err := r.createRoleBindingIfMissing(ctx, runnerObject, log); err != nil {
		return err
	}
	return nil
}

func (r *RunnerReconciler) getGitlabApiClient(ctx context.Context, runnerObject *gitlabv1beta1.Runner) (api.GitlabClient, error) {
	// if the client is already defined, return that one instead of trying to obtain a new one.
	if r.GitlabApiClient != nil {
		return r.GitlabApiClient, nil
	}

	// if we have defined token in the config, then use that one.
	if runnerObject.Spec.RegistrationConfig.Token != nil && *runnerObject.Spec.RegistrationConfig.Token != "" {
		return api.NewGitlabClient(*runnerObject.Spec.RegistrationConfig.Token, runnerObject.Spec.GitlabInstanceURL)
	}

	// we did not store the registration token in clear view. Hopefully we have defined and created a secret holding it
	if runnerObject.Spec.RegistrationConfig.TokenSecret == "" {
		return nil, coreErrors.New("you need to either define a token or a secret pointing to it")
	}

	// let's try to fetch the secret with our token
	var gitlabSecret corev1.Secret
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: runnerObject.Namespace,
		Name:      runnerObject.Spec.RegistrationConfig.TokenSecret,
	}, &gitlabSecret); err != nil {
		return nil, err
	}

	//
	token, ok := gitlabSecret.Data["token"]
	if !ok || string(token) == "" {
		return nil, coreErrors.New("secret doesn't contain field token or it's empty")
	}

	// and finally
	decryptedToken := string(token)
	runnerObject.Spec.RegistrationConfig.Token = &decryptedToken
	return api.NewGitlabClient(*runnerObject.Spec.RegistrationConfig.Token, runnerObject.Spec.GitlabInstanceURL)
}

// RegisterNewRunnerOnGitlab registers runner against gitlab server and saves the value inside the status
func (r *RunnerReconciler) RegisterNewRunnerOnGitlab(ctx context.Context, runner *gitlabv1beta1.Runner, log logr.Logger) (ctrl.Result, error) {
	// get the gitlab runner
	gitlabApiClient, err := r.getGitlabApiClient(ctx, runner)
	if err != nil {
		return ctrl.Result{}, err
	}

	// obtain the registration token from gitlab
	token, err := gitlabApiClient.Register(runner.Spec.RegistrationConfig)
	if err != nil {
		log.Error(err, "cannot register the runner against gitlab api")
		runner.Status.Error = err.Error()
		if errUpdate := r.Client.Status().Update(ctx, runner); errUpdate != nil {
			log.Error(errUpdate, "cannot set the status of the runner object")
		}
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

	// set the new auth token and record the reg details used for the operation (token and tags)
	runner.Status.Error = ""
	runner.Status.AuthenticationToken = token
	if err = r.Client.Status().Update(ctx, runner); err != nil {
		log.Error(err, "cannot update runner status")
		return resultRequeueAfterDefaultTimeout, err
	}
	// all is fine, new token has been applied, requeue the runner and check create/amend a deployment if needed
	newRunner := runner.DeepCopy()

	// check if annotations has been properly initialized
	newRunner.Annotations[lastRegistrationTokenAnnotationKey] = *runner.Spec.RegistrationConfig.Token
	newRunner.Annotations[lastRegistrationTags] = strings.Join(runner.Spec.RegistrationConfig.TagList, ",")
	if err = r.Client.Update(ctx, newRunner); err != nil {
		log.Error(err, "cannot update runner with authentication token")
		return resultRequeueAfterDefaultTimeout, err
	}

	log.Info("registered a new runner on gitlab server")
	return ctrl.Result{Requeue: true}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// setup indexers on configmap created by x
	ctx := context.Background()
	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.ConfigMap{}, ownerCmKey, func(object client.Object) []string {
		// grab the configMap object, extract the owner...
		configMap := object.(*corev1.ConfigMap)
		owner := metav1.GetControllerOf(configMap)
		if owner == nil {
			return nil
		}

		// ensure that we dealing with a proper object
		if owner.APIVersion != gitlabv1beta1.GroupVersion.String() || owner.Kind != "Runner" {
			return nil
		}

		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}
	// deployments
	// todo : unify with configmap above
	if err := mgr.GetFieldIndexer().IndexField(ctx, &appsv1.Deployment{}, ownerDpKey, func(rawObj client.Object) []string {
		// grab the deployment object, extract the owner...
		deployment := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}

		// ensure that we dealing with a proper object
		if owner.APIVersion != gitlabv1beta1.GroupVersion.String() || owner.Kind != "Runner" {
			return nil
		}

		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitlabv1beta1.Runner{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				if e.ObjectOld == nil || e.ObjectNew == nil {
					return false
				}
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
			DeleteFunc: func(event event.DeleteEvent) bool {
				// The reconciler adds a finalizer when the delete timestamp is added.
				// Avoid reconciling in case it's a runner, we still want to reconcile if it's a dependent object
				if _, ok := event.Object.(*gitlabv1beta1.Runner); ok {
					return false
				}
				return true
			}}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
