/*


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
	gitlabRunOp "go.alekc.dev/gitlab-runner-operator/api/v1alpha1"
	internalApi "go.alekc.dev/gitlab-runner-operator/internal/api"
	"go.alekc.dev/gitlab-runner-operator/internal/crypto"
	interlalErrors "go.alekc.dev/gitlab-runner-operator/internal/errors"
	"go.alekc.dev/gitlab-runner-operator/internal/generate"
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
)

const ownerCmKey = ".metadata.cm.controller"
const ownerDpKey = ".metadata.dp.controller"

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
	GitlabApiClient internalApi.GitlabClient
}

// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=runners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=runners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=deployments;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=*
// +kubebuilder:rbac:groups=core,resources=serviceaccount,verbs=get;create;delete

func (r *RunnerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("runner", req.NamespacedName)
	resultRequeueAfterDefaultTimeout := ctrl.Result{Requeue: true, RequeueAfter: defaultTimeout}

	// find the object, in case we cannot find it just return.
	runnerObj := &gitlabRunOp.Runner{}
	err := r.Client.Get(ctx, req.NamespacedName, runnerObj)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Sanity check. Initialize our annotations if needed.
	if runnerObj.Annotations == nil {
		runnerObj.Annotations = make(map[string]string, 0)
	}

	// create required rbac credentials
	err = r.CreateRBACIfMissing(ctx, runnerObj, logger)
	if err != nil {
		return ctrl.Result{RequeueAfter: defaultTimeout}, err
	}

	// if the runner doesn't have a saved authentication token or the latest registration token/tags are different from the
	// current one, we need to redo the registration
	if runnerObj.Status.AuthenticationToken == "" ||
		runnerObj.GetAnnotation(lastRegistrationTokenAnnotationKey) != *runnerObj.Spec.RegistrationConfig.Token ||
		runnerObj.GetAnnotation(lastRegistrationTags) != strings.Join(runnerObj.Spec.RegistrationConfig.TagList, ",") {
		return r.RegisterNewRunnerOnGitlab(ctx, runnerObj, logger)
	}

	// create a config object
	textualConfigMap, err := generate.ConfigText(runnerObj)
	if err != nil {
		logger.Error(err, "cannot generate toml config")
		return resultRequeueAfterDefaultTimeout, err
	}

	// persist the configMap version in the runner status
	configMapVersion := crypto.StringToSHA1(textualConfigMap)
	if runnerObj.Status.ConfigMapVersion != configMapVersion {
		logger.Info("a new version of config map detected. updating Runner",
			"new_version", configMapVersion, "old_version", runnerObj.Status.ConfigMapVersion)
		runnerObj.Status.ConfigMapVersion = configMapVersion
		err = r.Client.Update(ctx, runnerObj)
		if err != nil {
			logger.Error(err, "cannot assign config map hash to the runner")
			return resultRequeueAfterDefaultTimeout, err
		}
	}

	var cm *corev1.ConfigMap
	var configMaps corev1.ConfigMapList

	// fetch children configmap
	err = r.Client.List(
		ctx,
		&configMaps,
		client.InNamespace(runnerObj.Namespace),
		client.MatchingFields{ownerCmKey: string(runnerObj.UID)},
	)
	if err != nil {
		logger.Error(err, "cannot list dependent configmaps")
		return ctrl.Result{}, err
	}

	// try to find one with the same name (in which case save it), and delete everything else (in case we have renamed)
	// our name in the config
	for _, k8Cm := range configMaps.Items {
		if k8Cm.Name == runnerObj.ChildName() {
			// we found our dependent config map, save it
			tmp := k8Cm
			cm = &tmp
			continue
		}
		logger.Info("deleting obsolete config map", "configMapName", k8Cm.Name)
		err = r.Delete(ctx, k8Cm.DeepCopy())
		if err != nil {
			// log the error but pretty much ignore it for now.
			logger.Error(err, "cannot delete obsolete config map", "zombie-name", k8Cm.Name)
		}
	}

	// if the config map doesn't exist, create it
	if cm == nil {
		cm = &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:            runnerObj.ChildName(),
				Namespace:       runnerObj.Namespace,
				OwnerReferences: runnerObj.GenerateOwnerReference(),
				Annotations:     map[string]string{configVersionAnnotationKey: configMapVersion},
			},
			Data: map[string]string{configMapKeyName: textualConfigMap},
		}
		logger.Info("creating config map object", "configMapName", cm.Name)
		err = r.Client.Create(ctx, cm)
		if err != nil {
			logger.Error(err, "cannot create a config map", "configMapName", cm.Name)
			return resultRequeueAfterDefaultTimeout, err
		}
		// config map has been created, requeue to ensure the proper state
		return ctrl.Result{Requeue: true}, nil
	}

	// ensure that our configMap has the same data
	if value, ok := cm.Data[configMapKeyName]; !ok || value != textualConfigMap {
		logger.Info("config map object has old config, needs updating", "configMapName", cm.Name)
		newObj := cm.DeepCopy()
		newObj.Data[configMapKeyName] = textualConfigMap
		newObj.Annotations[configVersionAnnotationKey] = configMapVersion
		err = r.Update(ctx, newObj)
		if err != nil && !interlalErrors.IsStale(err) {
			logger.Error(
				err,
				"cannot update config map with the new configuration",
				"config_map_name", cm.Name)
			return ctrl.Result{Requeue: true}, err
		}
		// all is good, reconcile this runner again
		return ctrl.Result{Requeue: true}, nil
	}

	// fetch related deployment
	var deployments appsv1.DeploymentList
	err = r.Client.List(
		ctx,
		&deployments,
		client.InNamespace(runnerObj.Namespace),
		client.MatchingFields{ownerDpKey: string(runnerObj.UID)},
	)
	if err != nil {
		logger.Error(err, "cannot list dependent deployments")
		return ctrl.Result{}, err
	}

	// try to find one with the same name (in which case save it), and delete everything else (in case we have renamed)
	// our name in the config
	deploymentName := runnerObj.ChildName()
	var deployment *appsv1.Deployment
	for _, dp := range deployments.Items {
		if dp.Name == deploymentName {
			// we found our dependent config map, save it
			tmp := dp
			deployment = &tmp
			continue
		}
		logger.Info("found obsolete deployment, deleting it", "deployment_name", dp.Name)
		err = r.Delete(ctx, dp.DeepCopy())
		if err != nil {
			// log the error but pretty much ignore it for now.
			logger.Error(err, "cannot delete zombie deployment", "deployment_name", dp.Name)
		}
	}

	labels := map[string]string{"deployment": runnerObj.Name}
	newDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
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
						Image:           "gitlab/gitlab-runner:latest", //todo: param
						Resources:       corev1.ResourceRequirements{}, //todo:
						LivenessProbe:   nil,
						ReadinessProbe:  nil,
						ImagePullPolicy: "Always", //todo
						SecurityContext: nil,
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/etc/gitlab-runner/",
						}},
					}},
					EphemeralContainers:           nil,
					RestartPolicy:                 "",
					TerminationGracePeriodSeconds: nil,
					ActiveDeadlineSeconds:         nil,
					DNSPolicy:                     "",
					NodeSelector:                  nil,
					ServiceAccountName:            runnerObj.ChildName(),
					AutomountServiceAccountToken:  nil,
					NodeName:                      "",
					HostNetwork:                   false,
					HostPID:                       false,
					HostIPC:                       false,
					ShareProcessNamespace:         nil,
					SecurityContext:               nil,
					ImagePullSecrets:              nil,
					Hostname:                      "",
					Subdomain:                     "",
					Affinity:                      nil,
					SchedulerName:                 "",
					Tolerations:                   nil,
					HostAliases:                   nil,
					PriorityClassName:             "",
					Priority:                      nil,
					DNSConfig:                     nil,
					ReadinessGates:                nil,
					RuntimeClassName:              nil,
					EnableServiceLinks:            nil,
					PreemptionPolicy:              nil,
					Overhead:                      nil,
					TopologySpreadConstraints:     nil,
				},
			},
			Strategy:                appsv1.DeploymentStrategy{},
			MinReadySeconds:         0,
			RevisionHistoryLimit:    nil,
			Paused:                  false,
			ProgressDeadlineSeconds: nil,
		},
	}
	if deployment == nil {
		logger.Info("creating a new deployment", "deployment_name", newDeployment.Name)
		err = r.Client.Create(ctx, &newDeployment)
		if err != nil {
			logger.Error(err, "cannot create the deployment")
			return resultRequeueAfterDefaultTimeout, err
		}
		// deployment has been created, requeue to proceed further
		return ctrl.Result{Requeue: true}, nil
	}

	// check if there are any differences between 2 deployments
	var existingConfigMap string
	if val, ok := deployment.GetAnnotations()[configVersionAnnotationKey]; ok {
		existingConfigMap = val
	}
	if existingConfigMap != runnerObj.Status.ConfigMapVersion || !equality.Semantic.DeepDerivative(newDeployment.Spec, deployment.DeepCopy().Spec) {
		logger.Info("deployment is different from our version, updating", "deployment_name", newDeployment.Name)
		err = r.Client.Update(ctx, &newDeployment)
		if err != nil {
			logger.Error(err, "cannot update deployment")
			return resultRequeueAfterDefaultTimeout, err
		}
	}

	// generate the map
	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) CreateRBACIfMissing(ctx context.Context, runnerObject *gitlabRunOp.Runner, log logr.Logger) error {
	// create default service account
	// todo: deal with renames of the runner
	runnerName := runnerObject.ChildName()
	namespacedKey := client.ObjectKey{Namespace: runnerObject.Namespace, Name: runnerName}
	err := r.Client.Get(ctx, namespacedKey, &corev1.ServiceAccount{})
	if err != nil {
		// if the error is different from not found (which is acceptable for us), then return from the function
		if !errors.IsNotFound(err) {
			log.Error(err, "cannot get the service account")
			return err
		}

		// by this point we do not have any sa. Create one
		log.Info("creating sa")
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:            runnerName,
				Namespace:       runnerObject.Namespace,
				OwnerReferences: runnerObject.GenerateOwnerReference(),
			},
		}
		err = r.Client.Create(ctx, sa)
		if err != nil {
			log.Error(err, "cannot create service-account")
			return err
		}
	}

	// check if we need to create a role
	err = r.Client.Get(ctx, namespacedKey, &v1.Role{})
	if err != nil {
		// if the error is different from not found (which is acceptable for us), then return from the function
		if !errors.IsNotFound(err) {
			log.Error(err, "cannot get the role")
			return err
		}

		// by this point we do not have any role. Create one
		log.Info("creating the role")
		role := &v1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:            runnerName,
				Namespace:       runnerObject.Namespace,
				OwnerReferences: runnerObject.GenerateOwnerReference(),
			},
			Rules: []v1.PolicyRule{{
				Verbs:     []string{"get", "list", "watch", "create", "patch", "delete"},
				APIGroups: []string{"*"},
				Resources: []string{"pods", "pods/exec", "secrets"},
			}},
		}
		err = r.Client.Create(ctx, role)
		if err != nil {
			log.Error(err, "cannot create role")
			return err
		}
	}

	// check if we need to create the binding
	err = r.Client.Get(ctx, namespacedKey, &v1.RoleBinding{})
	if err != nil {
		// if the error is different from not found (which is acceptable for us), then return from the function
		if !errors.IsNotFound(err) {
			log.Error(err, "cannot get the rolebinding")
			return err
		}

		// by this point we do not have any role. Create one
		log.Info("creating the rolebinding")
		role := &v1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:            runnerName,
				Namespace:       runnerObject.Namespace,
				OwnerReferences: runnerObject.GenerateOwnerReference(),
			},
			Subjects: []v1.Subject{{
				Kind:      "ServiceAccount",
				Name:      runnerName,
				Namespace: runnerObject.Namespace,
			}},
			RoleRef: v1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     runnerName,
			},
		}
		err = r.Client.Create(ctx, role)
		if err != nil {
			log.Error(err, "cannot create rolebinding")
			return err
		}
	}
	return nil
}

func (r *RunnerReconciler) getGitlabApiClient(ctx context.Context, runnerObject *gitlabRunOp.Runner) (internalApi.GitlabClient, error) {
	// if the client is already defined, return that one instead of trying to obtain a new one.
	if r.GitlabApiClient != nil {
		return r.GitlabApiClient, nil
	}

	// if we have defined token in the config, then use that one.
	if runnerObject.Spec.RegistrationConfig.Token != nil && *runnerObject.Spec.RegistrationConfig.Token != "" {
		return internalApi.NewGitlabClient(*runnerObject.Spec.RegistrationConfig.Token, runnerObject.Spec.GitlabInstanceURL)
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
	return internalApi.NewGitlabClient(*runnerObject.Spec.RegistrationConfig.Token, runnerObject.Spec.GitlabInstanceURL)
}
func (r *RunnerReconciler) RegisterNewRunnerOnGitlab(ctx context.Context, runner *gitlabRunOp.Runner, log logr.Logger) (ctrl.Result, error) {
	// register the client against the gitlab api and obtain the authentication token
	gitlabApiClient, err := r.getGitlabApiClient(ctx, runner)
	if err != nil {
		return ctrl.Result{}, err
	}
	token, err := gitlabApiClient.Register(runner.Spec.RegistrationConfig)
	if err != nil {
		log.Error(err, "cannot register the runner against gitlab api")
		runner.Status.Error = err.Error()
		errUpdate := r.Client.Update(ctx, runner)
		if errUpdate != nil {
			log.Error(errUpdate, "cannot set the status of the runner object")
		}
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

	// all is fine, new token has been applied, requeue the runner and check create/amend a deployment if needed
	newRunner := runner.DeepCopy()
	newRunner.Status.Error = ""

	// set the new auth token and record the reg details used for the operation (token and tags)
	newRunner.Status.AuthenticationToken = token
	newRunner.GetAnnotations()
	// check if annotations has been properly initialized
	newRunner.Annotations[lastRegistrationTokenAnnotationKey] = *runner.Spec.RegistrationConfig.Token
	newRunner.Annotations[lastRegistrationTags] = strings.Join(runner.Spec.RegistrationConfig.TagList, ",")
	err = r.Client.Update(ctx, newRunner)
	if err != nil {
		log.Error(err, "cannot update runner with authentication token")
		return ctrl.Result{Requeue: true, RequeueAfter: defaultTimeout}, err
	}
	log.Info("registered a new runner on gitlab server")
	return ctrl.Result{Requeue: true}, err
}

func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// setup indexers on configmap created by x
	if err := mgr.GetFieldIndexer().IndexField(&corev1.ConfigMap{}, ownerCmKey, func(rawObj runtime.Object) []string {
		// grab the configMap object, extract the owner...
		configMap := rawObj.(*corev1.ConfigMap)
		owner := metav1.GetControllerOf(configMap)
		if owner == nil {
			return nil
		}

		// ensure that we dealing with a proper object
		if owner.APIVersion != gitlabRunOp.GroupVersion.String() || owner.Kind != "Runner" {
			return nil
		}

		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}
	// deployments
	// todo : unify with configmap above
	if err := mgr.GetFieldIndexer().IndexField(&appsv1.Deployment{}, ownerDpKey, func(rawObj runtime.Object) []string {
		// grab the deployment object, extract the owner...
		deployment := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}

		// ensure that we dealing with a proper object
		if owner.APIVersion != gitlabRunOp.GroupVersion.String() || owner.Kind != "Runner" {
			return nil
		}

		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitlabRunOp.Runner{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
