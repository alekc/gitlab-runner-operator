package validate

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/internal/generate"
	"gitlab.k8s.alekc.dev/internal/result"
	"gitlab.k8s.alekc.dev/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Deployment(ctx context.Context, cl client.Client, runnerObj types.RunnerInfo, logger logr.Logger) (*ctrl.Result, error) {
	labels := map[string]string{"deployment": runnerObj.GetName()}
	// gitlab-runner reads system_id only from a file next to config.toml, so it
	// is carried as a pod annotation and projected into the config volume.
	// Read-only is enough: the runner skips its save path, and the warning it
	// logs on failure, whenever the file already parses.
	systemID := generate.SystemID(runnerObj.GetUID())
	wantedDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runnerObj.ChildName(),
			Namespace: runnerObj.GetNamespace(),
			Annotations: map[string]string{
				types.ConfigVersionAnnotationKey: runnerObj.ConfigMapVersion(),
				types.SystemIDAnnotationKey:      systemID,
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
						types.SystemIDAnnotationKey:      systemID,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector:      runnerObj.RunnerNodeSelector(),
					Tolerations:       runnerObj.RunnerTolerations(),
					Affinity:          runnerObj.RunnerAffinity(),
					PriorityClassName: runnerObj.RunnerPriorityClassName(),
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: runnerObj.ChildName(),
										},
									}},
									{DownwardAPI: &corev1.DownwardAPIProjection{
										Items: []corev1.DownwardAPIVolumeFile{{
											Path: types.SystemIDFileName,
											FieldRef: &corev1.ObjectFieldSelector{
												FieldPath: "metadata.annotations['" + types.SystemIDAnnotationKey + "']",
											},
										}},
									}},
								},
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:            types.RunnerContainerName,
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

	// Deployment exists. Roll it only on a tracked change: the rendered config,
	// the derived system_id, or the manager pod shape. The system_id is read
	// from the pod template, the copy the projection resolves, so a rollback to
	// a revision without it is repaired instead of silently kept.
	//
	// The shape compare is a NAMED subset, never the whole spec: the apiserver
	// defaults fields we never set (port protocol, terminationMessagePath), so
	// a whole-spec diff never converges and the controller re-applies its
	// sparse spec forever, never reaching Ready. The subset is not immune to
	// that either, which is what the post-update check below handles.
	existingSystemID := existingDeployment.Spec.Template.GetAnnotations()[types.SystemIDAnnotationKey]
	if existingDeployment.GetAnnotations()[types.ConfigVersionAnnotationKey] != runnerObj.ConfigMapVersion() ||
		existingSystemID != systemID ||
		!apiequality.Semantic.DeepEqual(
			ManagerPodShape(&existingDeployment.Spec.Template),
			ManagerPodShape(&wantedDeployment.Spec.Template)) {
		logger.Info("deployment changed (config, system id or pod shape), updating", "deployment_name", existingDeployment.Name)
		intendedShape := ManagerPodShape(&wantedDeployment.Spec.Template)
		// Carry the live ResourceVersion so this is a conditional update: a
		// concurrent change conflicts and requeues instead of being silently
		// overwritten by our from-scratch spec.
		wantedDeployment.ResourceVersion = existingDeployment.ResourceVersion
		err = cl.Update(ctx, &wantedDeployment)
		if err != nil {
			logger.Error(err, "cannot update deployment")
			return result.RequeueWithDefaultTimeout(), err
		}
		// Update decodes the stored object back into wantedDeployment, so this
		// compares what the cluster kept against what we asked for. A gap means
		// re-applying cannot close it: either the apiserver dropped a field this
		// CRD accepts, or a mutating webhook rewrote ours. Settle instead of
		// requeueing forever and never reporting the runner ready.
		storedShape := ManagerPodShape(&wantedDeployment.Spec.Template)
		if !apiequality.Semantic.DeepEqual(intendedShape, storedShape) {
			logger.Info("cluster did not store the manager pod spec as sent; leaving it as stored",
				"deployment_name", existingDeployment.Name,
				"intended", shapeJSON(intendedShape),
				"stored", shapeJSON(storedShape))
			return nil, nil
		}
		return result.RequeueNow(), nil
	}
	return nil, nil
}

// Secret renders the runner config.toml into an Opaque Secret named after the
// runner, keeping each entry's authentication token under a dedicated
// ConfigTokenKeyPrefix key so the controller can recover it on later
// reconciles. The token never lands in a ConfigMap or in the CR status.
func Secret(ctx context.Context, cl client.Client, runnerObj types.RunnerInfo, logger logr.Logger, gitlabRunnerTomlConfig string, tokens map[string]string, caPEM []byte) (*ctrl.Result, error) {
	desired := map[string][]byte{
		types.ConfigMapKeyName: []byte(gitlabRunnerTomlConfig),
	}
	for name, token := range tokens {
		desired[types.ConfigTokenKeyPrefix+name] = []byte(token)
	}
	// When a custom CA is configured, store it alongside config.toml so it is
	// mounted into the runner at types.CACertFile (config.toml's tls-ca-file).
	if len(caPEM) > 0 {
		desired[types.CACertFileName] = caPEM
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
