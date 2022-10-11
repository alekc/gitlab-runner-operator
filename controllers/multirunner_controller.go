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

	"gitlab.k8s.alekc.dev/internal/api"
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
	Scheme          *runtime.Scheme
	GitlabApiClient api.GitlabClient
}

// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=multirunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=multirunners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=multirunners/finalizers,verbs=update
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="core",resources=configmaps;secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=*

func (r *MultiRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

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
