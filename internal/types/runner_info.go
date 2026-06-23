package types

import (
	"context"

	"gitlab.k8s.alekc.dev/api/v1beta2"
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
	RegistrationConfig() []v1beta2.GitlabRegInfo
	StoreRunnerRegistration(v1beta2.GitlabRegInfo)
}
