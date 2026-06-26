package types

import (
	"context"

	"gitlab.k8s.alekc.dev/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RunnerInfo interface {
	// client.Object gives the metav1.Object + runtime.Object surface
	// (name, namespace, annotations, deepcopy) used generically by the
	// reconcilers (status writes, the delete-attempt annotation).
	client.Object

	ChildName() string
	GetStatus() any

	GenerateOwnerReference() []v1.OwnerReference
	IsBeingDeleted() bool
	HasFinalizer() bool
	RemoveFinalizer()
	AddFinalizer() (finalizerUpdated bool)
	Update(ctx context.Context, writer client.Writer) error
	SetStatusError(errorMessage string)
	SetConfigMapVersion(versionHash string)
	SetStatus(newStatus any)
	UpdateStatus(ctx context.Context, writer client.StatusWriter) error
	SetStatusReady(ready bool)
	ConfigMapVersion() string
	RunnerImage() string
	// CACertificate returns the custom CA source, or nil when none is set.
	CACertificate() *v1beta2.CASource
	RunnerResources() corev1.ResourceRequirements
	RunnerImagePullPolicy() corev1.PullPolicy
	RunnerSecurityContext() *corev1.SecurityContext
	SetObservedGeneration(generation int64)
	SetReadyCondition(ready bool, reason, message string)
	RegistrationConfig() []v1beta2.GitlabRegInfo
	StoreRunnerRegistration(v1beta2.GitlabRegInfo)
	// ExecutorConfigs returns the kubernetes executor config for each runner
	// unit (one for a Runner, one per entry for a MultiRunner). Used to derive
	// the build namespaces the runner ServiceAccount needs RBAC in.
	ExecutorConfigs() []*v1beta2.KubernetesConfig
}
