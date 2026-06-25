/*
Copyright 2020 Alexander Chernov

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

package controller

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/internal/api"
	"gitlab.k8s.alekc.dev/internal/crud"
	"gitlab.k8s.alekc.dev/internal/generate"
	"gitlab.k8s.alekc.dev/internal/result"
	"gitlab.k8s.alekc.dev/internal/validate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
)

const defaultTimeout = 15 * time.Second
const configMapKeyName = "config.toml"
const configVersionAnnotationKey = "config-version"

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	// APIReader is the uncached reader (mgr.GetAPIReader); used for reads of
	// resources the operator does not watch, to avoid spinning up their
	// cluster-wide informers (e.g. the shared executor ClusterRole).
	APIReader       client.Reader
	Log             logr.Logger
	Scheme          *runtime.Scheme
	GitlabApiClient api.GitlabClient
}

var resultRequeueAfterDefaultTimeout = ctrl.Result{Requeue: true, RequeueAfter: defaultTimeout}
var resultRequeueNow = ctrl.Result{Requeue: true}

// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=runners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=runners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gitlab.k8s.alekc.dev,resources=runners/finalizers,verbs=update
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="core",resources=configmaps;secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch

// The operator maintains one shared ClusterRole holding the kubernetes executor
// permission set and binds each runner ServiceAccount to it. It has no rbac
// "escalate" verb, so it must itself hold these permissions both to write that
// ClusterRole and to bind runners to it on clusters that enforce RBAC
// escalation prevention. This is the explicit ceiling for what a runner can be
// granted.
// +kubebuilder:rbac:groups="core",resources=pods,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups="core",resources=pods/exec;pods/attach,verbs=get;create;patch;delete
// +kubebuilder:rbac:groups="core",resources=pods/log,verbs=get;list
// +kubebuilder:rbac:groups="core",resources=services,verbs=get;create
// +kubebuilder:rbac:groups="core",resources=events,verbs=list;watch

func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// find the object, in case we cannot find it just return.
	runnerObj, err := crud.SingleRunner(ctx, r.Client, req.NamespacedName)
	if err != nil {
		return *result.DontRequeue(), client.IgnoreNotFound(err)
	}

	logger.Info("reconciling")
	if runnerObj.IsBeingDeleted() {
		return finalizeDeletion(ctx, r.Client, r.APIReader, r.GitlabApiClient, runnerObj, logger)
	}

	// update the status when done processing in case there is anything pending
	defer func() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			newRunner, err := crud.SingleRunner(
				context.Background(),
				r.Client,
				client.ObjectKey{
					Namespace: runnerObj.GetNamespace(),
					Name:      runnerObj.GetName()},
			)
			switch {
			case err != nil:
				logger.Error(err, "cannot get runner")
				return err

			// no changes in status detected
			case reflect.DeepEqual(runnerObj.GetStatus(), newRunner.GetStatus()):
				return nil
			}
			newRunner.SetStatus(runnerObj.GetStatus())
			return newRunner.UpdateStatus(ctx, r.Status())
		})
		if err != nil {
			logger.Error(err, "cannot update runner's status")
		}
	}()

	// if finalizer is not registered, do it now
	if !runnerObj.HasFinalizer() {
		logger.Info("setting finalizer")
		runnerObj.AddFinalizer()
		if err := runnerObj.Update(ctx, r); err != nil {
			logger.Error(err, "cannot set finalizer")
			return resultRequeueAfterDefaultTimeout, err
		}
		return resultRequeueNow, nil
	}

	// reset the error; it is re-set below if anything fails this pass
	runnerObj.SetStatusError("")
	runnerObj.SetStatusReady(false)
	runnerObj.SetObservedGeneration(runnerObj.GetGeneration())

	// resolve auth and ensure managed runners exist on GitLab. Managed runner
	// ids are persisted to status immediately inside ensureRunners.
	tokens, requeueAfter, err := ensureRunners(ctx, r.Client, r.Status(), r.GitlabApiClient, runnerObj, logger)
	if err != nil {
		runnerObj.SetStatusError(err.Error())
		runnerObj.SetReadyCondition(false, "AuthFailed", err.Error())
		logger.Error(err, "cannot ensure runners against gitlab")
		return resultRequeueAfterDefaultTimeout, err
	}

	// create required rbac credentials if they are missing
	if err = crud.CreateRBACIfMissing(ctx, r.Client, r.APIReader, runnerObj, logger); err != nil {
		runnerObj.SetStatusError("Cannot create the rbac objects")
		runnerObj.SetReadyCondition(false, "RBACFailed", err.Error())
		logger.Error(err, "cannot create rbac objects")
		return resultRequeueAfterDefaultTimeout, err
	}

	// render config.toml from the resolved tokens
	generatedTomlConfig, configHashKey, err := generate.TomlConfig(runnerObj, tokens)
	if err != nil {
		runnerObj.SetStatusError(err.Error())
		runnerObj.SetReadyCondition(false, "ConfigRenderFailed", err.Error())
		logger.Error(err, "cannot generate runner config")
		return resultRequeueAfterDefaultTimeout, err
	}

	// Record the freshly rendered config hash on status every reconcile. It is
	// the single source of truth for the status version field and the
	// deployment's config-version annotation. Deriving both from the live hash
	// (not a value re-read from the cache) avoids a race where a rapid requeue
	// reads stale status, skips the deployment roll, reverts the version, and
	// leaves the runner stuck below Ready.
	runnerObj.SetConfigMapVersion(configHashKey)

	// resolve the custom CA bundle (inline value, Secret, or ConfigMap) so it is
	// stored in the config Secret and trusted by the runner pod.
	caPEM, err := resolveCABundle(ctx, r.Client, runnerObj.GetNamespace(), runnerObj.CACertificate())
	if err != nil {
		runnerObj.SetReadyCondition(false, "CAResolveFailed", err.Error())
		return resultRequeueAfterDefaultTimeout, err
	}

	// reconcile the config Secret (config.toml, the per-entry tokens, and the CA)
	if res, err := validate.Secret(ctx, r.Client, runnerObj, logger, generatedTomlConfig, tokens, caPEM); res != nil || err != nil {
		if err != nil {
			runnerObj.SetReadyCondition(false, "ConfigSecretFailed", err.Error())
		}
		return *res, err
	}

	// validate deployment data
	if res, err := validate.Deployment(ctx, r.Client, runnerObj, logger); res != nil || err != nil {
		if err != nil {
			runnerObj.SetReadyCondition(false, "DeploymentFailed", err.Error())
		}
		return *res, err
	}

	runnerObj.SetStatusReady(true)
	runnerObj.SetReadyCondition(true, "Reconciled", "runner is ready")
	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	return *result.DontRequeue(), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	const runnerOwnerSecretKey = ".metadata.secretcontroller"
	const runnerOwnerDpKey = ".metadata.dpcontroller"
	ctx := context.Background()
	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.Secret{}, runnerOwnerSecretKey, func(object client.Object) []string {
		// grab the secret object, extract the owner...
		secret := object.(*corev1.Secret)
		owner := metav1.GetControllerOf(secret)
		if owner == nil {
			return nil
		}

		// ensure that we're dealing with a proper object
		if owner.APIVersion != gitlabv1beta2.GroupVersion.String() || owner.Kind != "Runner" {
			return nil
		}

		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}
	// deployments
	// todo : unify with configmap above
	if err := mgr.GetFieldIndexer().IndexField(ctx, &appsv1.Deployment{}, runnerOwnerDpKey, func(rawObj client.Object) []string {
		// grab the deployment object, extract the owner...
		deployment := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}

		// ensure that we're dealing with a proper object
		if owner.APIVersion != gitlabv1beta2.GroupVersion.String() || owner.Kind != "Runner" {
			return nil
		}

		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		// Only the primary object gets the generation filter: it avoids a
		// reconcile loop on our own status writes, and the actual delete event
		// is ignored because deletion is finalizer-driven. Owned objects must
		// NOT inherit this filter, or edits to them (which do not bump
		// generation on Secret/RoleBinding/ServiceAccount) would never self-heal.
		For(&gitlabv1beta2.Runner{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				if e.ObjectOld == nil || e.ObjectNew == nil {
					return false
				}
				// Reconcile when the object is being deleted so the finalizer
				// runs promptly: marking deletion sets deletionTimestamp but
				// does not bump generation, so the generation check below would
				// otherwise drop it (leaving the object stuck in Terminating).
				if !e.ObjectNew.GetDeletionTimestamp().IsZero() {
					return true
				}
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
			DeleteFunc: func(event.DeleteEvent) bool { return false },
		})).
		Owns(&corev1.Secret{}).
		// Deployment carries a generation, so filter to spec changes and skip
		// its frequent status-only updates; create/delete still reconcile.
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
