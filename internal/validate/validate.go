package validate

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/internal/result"
	"gitlab.k8s.alekc.dev/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
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
			Replicas: ptr.To[int32](1),
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
							Secret: &corev1.SecretVolumeSource{
								SecretName: runnerObj.ChildName(),
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "runner",
						Image:           runnerObj.RunnerImage(),
						ImagePullPolicy: runnerObj.RunnerImagePullPolicy(),
						Resources:       runnerObj.RunnerResources(),
						SecurityContext: runnerObj.RunnerSecurityContext(),
						Ports: []corev1.ContainerPort{{
							Name:          "metrics",
							ContainerPort: 9090,
						}},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(9090)},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       20,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(9090)},
							},
							InitialDelaySeconds: 30,
							PeriodSeconds:       30,
						},
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

	// Deployment exists. Roll it only when the rendered config changed (the
	// config-version annotation) or the runner image changed. We deliberately do
	// NOT diff the whole spec: the API server defaults fields we never set
	// (port protocol, terminationMessagePath, and so on), so a subset/derivative
	// compare never converges and the controller re-applies its sparse spec
	// forever, never reaching Ready.
	existingImage := ""
	if len(existingDeployment.Spec.Template.Spec.Containers) > 0 {
		existingImage = existingDeployment.Spec.Template.Spec.Containers[0].Image
	}
	if existingDeployment.GetAnnotations()[types.ConfigVersionAnnotationKey] != runnerObj.ConfigMapVersion() || existingImage != runnerObj.RunnerImage() {
		logger.Info("deployment changed (config or image), updating", "deployment_name", existingDeployment.Name)
		err = cl.Update(ctx, &wantedDeployment)
		if err != nil {
			logger.Error(err, "cannot update deployment")
			return result.RequeueWithDefaultTimeout(), err
		}
		return result.RequeueNow(), nil
	}
	return nil, nil
}

// Secret renders the runner config.toml into an Opaque Secret named after the
// runner, keeping each entry's authentication token under a dedicated
// ConfigTokenKeyPrefix key so the controller can recover it on later
// reconciles. The token never lands in a ConfigMap or in the CR status.
func Secret(ctx context.Context, cl client.Client, runnerObj types.RunnerInfo, logger logr.Logger, gitlabRunnerTomlConfig string, tokens map[string]string) (*ctrl.Result, error) {
	desired := map[string][]byte{
		types.ConfigMapKeyName: []byte(gitlabRunnerTomlConfig),
	}
	for name, token := range tokens {
		desired[types.ConfigTokenKeyPrefix+name] = []byte(token)
	}

	var secret corev1.Secret
	err := cl.Get(ctx, client.ObjectKey{
		Namespace: runnerObj.GetNamespace(),
		Name:      runnerObj.ChildName(),
	}, &secret)

	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "got an error while fetching config secret")
		return result.RequeueWithDefaultTimeout(), err
	}

	if err != nil && errors.IsNotFound(err) {
		// secret doesn't exist, create it and requeue
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            runnerObj.ChildName(),
				Namespace:       runnerObj.GetNamespace(),
				OwnerReferences: runnerObj.GenerateOwnerReference(),
			},
			Type: corev1.SecretTypeOpaque,
			Data: desired,
		}
		if err = cl.Create(ctx, &secret); err != nil {
			runnerObj.SetStatusError("cannot create config secret")
			logger.Error(err, "cannot create a config secret", "secretName", secret.Name)
			return result.RequeueWithDefaultTimeout(), err
		}
		return result.RequeueNow(), nil
	}

	// secret exists. Update it if the rendered content differs
	if !reflect.DeepEqual(secret.Data, desired) {
		logger.Info("config secret has old content, needs updating", "secretName", secret.Name)
		newObj := secret.DeepCopy()
		newObj.Data = desired
		if err = cl.Update(ctx, newObj); err != nil {
			const errMsg = "cannot update config secret with the new configuration"
			runnerObj.SetStatusError(errMsg)
			logger.Error(err, errMsg)
			return &ctrl.Result{Requeue: true}, err
		}
		return result.RequeueNow(), nil
	}
	return nil, nil
}
