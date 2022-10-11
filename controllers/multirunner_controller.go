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

	"gitlab.k8s.alekc.dev/internal/result"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gitlabv1beta1 "gitlab.k8s.alekc.dev/api/v1beta1"
)

// MultiRunnerReconciler reconciles a MultiRunner object
type MultiRunnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=multirunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=multirunners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=multirunners/finalizers,verbs=update
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="core",resources=configmaps;secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=*

func (r *MultiRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)
	// const finalizer = "gitlab.k8s.alekc.dev/finalizer-multi-runner"
	// logger := log.FromContext(ctx)
	//
	// // find the object, in case we cannot find it just return.
	// runnerObj := &gitlabv1beta1.MultiRunner{}
	// err := r.Client.Get(ctx, req.NamespacedName, runnerObj)
	// if err != nil {
	// 	return *result.DontRequeue(), client.IgnoreNotFound(err)
	// }
	//
	// logger.Info("reconciling")
	//
	// // check if the runner is already being removed
	// if !runnerObj.ObjectMeta.DeletionTimestamp.IsZero() {
	// 	logger.Info("runner is being deleted")
	//
	// 	// todo: delete all registrations
	// 	controllerutil.RemoveFinalizer(runnerObj, finalizer)
	// 	if err := r.Update(ctx, runnerObj); err != nil {
	// 		logger.Error(err, "cannot remove finalizer")
	// 		return resultRequeueAfterDefaultTimeout, err
	// 	}
	// 	return resultRequeueNow, nil
	// }
	//
	// // update the status when done processing in case there is anything pending
	// defer func() {
	// 	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
	// 		var newRunner gitlabv1beta1.MultiRunner
	// 		err = r.Client.Get(
	// 			ctx,
	// 			client.ObjectKey{Namespace: runnerObj.Namespace, Name: runnerObj.GetName()},
	// 			&newRunner)
	// 		switch {
	// 		case err != nil:
	// 			logger.Error(err, "cannot get multi runner")
	// 			return err
	//
	// 		// no changes in status detected
	// 		case reflect.DeepEqual(runnerObj.Status, newRunner.Status):
	// 			return nil
	// 		}
	// 		newRunner.Status = runnerObj.Status
	// 		return r.Status().Update(ctx, &newRunner)
	// 	})
	// 	if err != nil {
	// 		logger.Error(err, "cannot update runner's status")
	// 	}
	// }()
	//
	// // if finalizer is not registered, do it now
	// if !controllerutil.ContainsFinalizer(runnerObj, finalizer) {
	// 	logger.Info("setting finalizer")
	// 	controllerutil.AddFinalizer(runnerObj, finalizer)
	// 	if err = r.Update(ctx, runnerObj); err != nil {
	// 		logger.Error(err, "cannot set finalizer")
	// 		return resultRequeueAfterDefaultTimeout, err
	// 	}
	// 	return resultRequeueNow, nil
	// }
	//
	// // reset the error. If there is one still present we will get it later on
	// runnerObj.Status.Error = ""
	// runnerObj.Status.Ready = false
	//
	// // // if the runner doesn't have a saved authentication token
	// // // or the latest registration token/tags are different from the
	// // // current one, we need to redo the registration
	// // if runnerObj.Status.AuthenticationToken == "" ||
	// // 	runnerObj.Status.LastRegistrationToken != *runnerObj.Spec.RegistrationConfig.Token ||
	// // 	!reflect.DeepEqual(runnerObj.Status.LastRegistrationTags, runnerObj.Spec.RegistrationConfig.TagList) {
	// //
	// // 	// since we are doing a new registration, IF the runner already has an authentication token, delete it from gitlab server
	// // 	if runnerObj.Status.AuthenticationToken != "" {
	// // 		if res, err := r.RemoveRunnerFromGitlab(ctx, runnerObj, logger); res != nil {
	// // 			return *res, err
	// // 		}
	// // 	}
	// //
	// // 	return r.RegisterNewRunnerOnGitlab(ctx, runnerObj, logger)
	// // }
	// //
	// // // create required rbac credentials if they are missing
	// // if err = r.CreateRBACIfMissing(ctx, runnerObj, logger); err != nil {
	// // 	runnerObj.Status.Error = "Cannot create the rbac objects"
	// // 	logger.Error(err, "cannot create rbac objects")
	// // 	return resultRequeueAfterDefaultTimeout, err
	// // }
	//
	// // generate a new config map based on the runner spec
	// generatedTomlConfig, configHashKey, err := generate.MultiRunnerConfig(runnerObj)
	// if err != nil {
	// 	logger.Error(err, "cannot generate config map")
	// 	return resultRequeueAfterDefaultTimeout, err
	// }
	//
	// // if the config version differs, perform the upgrade
	// // set the status with a config map hash
	// if result, err := validate.ConfigMap(ctx, r.Client, runnerObj, logger, generatedTomlConfig, configHashKey); result != nil || err != nil {
	// 	return *result, err
	// }
	//
	// // validate deployment data
	// if result, err := validate.Deployment(ctx, r.Client, runnerObj, logger); result != nil || err != nil {
	// 	return *result, err
	// }
	//
	// runnerObj.Status.Ready = true
	return *result.DontRequeue(), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MultiRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.ConfigMap{}, runnerOwnerCmKey, func(object client.Object) []string {
		// grab the configMap object, extract the owner...
		configMap := object.(*corev1.ConfigMap)
		owner := metav1.GetControllerOf(configMap)
		if owner == nil {
			return nil
		}

		// ensure that we're dealing with a proper object
		if owner.APIVersion != gitlabv1beta1.GroupVersion.String() || owner.Kind != "MultiRunner" {
			return nil
		}

		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}

	// deployments
	if err := mgr.GetFieldIndexer().IndexField(ctx, &appsv1.Deployment{}, runnerOwnerDpKey, func(rawObj client.Object) []string {
		// grab the deployment object, extract the owner...
		deployment := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}

		// ensure that we're dealing with a proper object
		if owner.APIVersion != gitlabv1beta1.GroupVersion.String() || owner.Kind != "MultiRunner" {
			return nil
		}

		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&gitlabv1beta1.MultiRunner{}).
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
				if _, ok := event.Object.(*gitlabv1beta1.MultiRunner); ok {
					return false
				}
				return true
			}}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
