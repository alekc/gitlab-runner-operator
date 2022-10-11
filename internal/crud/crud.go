package crud

import (
	"context"

	gitlabv1beta1 "gitlab.k8s.alekc.dev/api/v1beta1"
	internalTypes "gitlab.k8s.alekc.dev/internal/types"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SingleRunner fetches single runner from k8s
func SingleRunner(ctx context.Context, client client.Client, nsName types.NamespacedName) (internalTypes.RunnerInfo, error) {
	runnerObj := &gitlabv1beta1.Runner{}
	err := client.Get(ctx, nsName, runnerObj)
	return runnerObj, err
}
