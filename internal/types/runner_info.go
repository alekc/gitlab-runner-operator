package types

import (
	"context"

	"gitlab.k8s.alekc.dev/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RunnerInfo interface {
	GetNamespace() string
	ChildName() string
	GetName() string
	GetStatus() any

	GenerateOwnerReference() []v1.OwnerReference
	IsBeingDeleted() bool
	IsAuthenticated() bool
	HasFinalizer() bool
	RemoveFinalizer()
	AddFinalizer() (finalizerUpdated bool)
	Update(ctx context.Context, writer client.Writer) error
	SetStatusError(errorMessage string)
	SetConfigMapVersion(versionHash string)
	SetStatus(newStatus any)
	UpdateStatus(ctx context.Context, writer client.StatusWriter) error
	SetStatusReady(ready bool)
	HasValidAuth() bool
	// RegisterOnGitlab(api.GitlabClient, logr.Logger) (ctrl.Result, error)
	// DeleteFromGitlab(apiClient api.GitlabClient, logger logr.Logger) error
	ConfigMapVersion() string
	RegistrationConfig() []v1beta1.RegisterNewRunnerOptions
	StoreRunnerRegistration(authToken string, config v1beta1.RegisterNewRunnerOptions)
	GitlabAuthTokens() []string
}
