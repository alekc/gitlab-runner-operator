package validate

import (
	"context"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/internal/result"
	"gitlab.k8s.alekc.dev/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Deployment(ctx context.Context, cl client.Client, runnerObj types.RunnerInfo, logger logr.Logger) (*ctrl.Result, error) {
	labels := map[string]string{"deployment": runnerObj.GetName()}
	wantedDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runnerObj.ChildName(),
			Namespace: runnerObj.GetNamespace(),
			Annotations: map[string]string{
				types.ConfigVersionAnnotationKey: runnerObj.ConfigMapVersion(),
			},
			OwnerReferences: runnerObj.GenerateOwnerReference(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						types.ConfigVersionAnnotationKey: runnerObj.ConfigMapVersion(),
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
						Image:           "gitlab/gitlab-runner:alpine-v15.9.1",
						Resources:       corev1.ResourceRequirements{}, // todo:
						ImagePullPolicy: "IfNotPresent",                // todo
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
	err := cl.Get(ctx, client.ObjectKey{
		Namespace: runnerObj.GetNamespace(),
		Name:      runnerObj.ChildName(),
	}, &existingDeployment)

	// if deployment doesn't exist, create it
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "could not obtain the deployment")
			return result.RequeueWithDefaultTimeout(), err
		}

		if err = cl.Create(ctx, &wantedDeployment); err != nil {
			logger.Error(err, "cannot create a deployment", "deploymentName", existingDeployment.Name)
			return result.RequeueWithDefaultTimeout(), err
		}
		return result.RequeueNow(), nil
	}

	// deployment exists. Check the configMap annotation
	if existingDeployment.GetAnnotations()[types.ConfigVersionAnnotationKey] != runnerObj.ConfigMapVersion() || !equality.Semantic.DeepDerivative(wantedDeployment.Spec, existingDeployment.DeepCopy().Spec) {
		logger.Info("deployment is different from our version, updating", "deployment_name", existingDeployment.Name)
		err = cl.Update(ctx, &wantedDeployment)
		if err != nil {
			logger.Error(err, "cannot update deployment")
			return result.RequeueWithDefaultTimeout(), err
		}
		return result.RequeueNow(), nil
	}
	return nil, nil
}

func ConfigMap(ctx context.Context, cl client.Client, runnerObj types.RunnerInfo, logger logr.Logger, gitlabRunnerTomlConfig string, configHashKey string) (*ctrl.Result, error) {
	// fetch current config map
	var configMap corev1.ConfigMap
	err := cl.Get(ctx, client.ObjectKey{
		Namespace: runnerObj.GetNamespace(),
		Name:      runnerObj.ChildName(),
	}, &configMap)

	// create config map if it doesn't exist (or report an error)
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "got an error while fetching config map")
		return result.RequeueWithDefaultTimeout(), err
	}

	if err != nil && errors.IsNotFound(err) {
		// configmap doesn't exist, create it and requeue
		configMap = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            runnerObj.ChildName(),
				Namespace:       runnerObj.GetNamespace(),
				OwnerReferences: runnerObj.GenerateOwnerReference(),
			},
			Data: map[string]string{types.ConfigMapKeyName: gitlabRunnerTomlConfig},
		}
		if err = cl.Create(ctx, &configMap); err != nil {
			runnerObj.SetStatusError("cannot create config map")
			logger.Error(err, "cannot create a config map", "configMapName", configMap.Name)
			return result.RequeueWithDefaultTimeout(), err
		}
		runnerObj.SetConfigMapVersion(configHashKey)
		return result.RequeueNow(), nil
	}

	// configmap exists. Check the value for the config map is different
	if configMap.Data[types.ConfigMapKeyName] != gitlabRunnerTomlConfig {
		logger.Info("config map object has old config, needs updating", "configMapName", configMap.Name)
		newObj := configMap.DeepCopy()
		newObj.Data[types.ConfigMapKeyName] = gitlabRunnerTomlConfig
		if err = cl.Update(ctx, newObj); err != nil {
			const errMsg = "cannot update config map with the new configuration"
			runnerObj.SetStatusError(errMsg)
			logger.Error(err, errMsg)
			return &ctrl.Result{Requeue: true}, err
		}
		runnerObj.SetConfigMapVersion(configHashKey)
		return result.RequeueNow(), nil
	}
	return nil, nil
}
