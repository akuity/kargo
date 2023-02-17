package argocd

import (
	"context"
	"fmt"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetApplication returns a pointer to the Argo CD Application resource
// specified by the namespace and name arguments. If no such resource is found,
// nil is returned instead.
func GetApplication(
	ctx context.Context,
	ctrlRuntimeClient client.Client,
	namespace string,
	name string,
) (*argocd.Application, error) {
	app := argocd.Application{}
	if err := ctrlRuntimeClient.Get(
		ctx,
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		&app,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Argo CD Application %q in namespace %q",
			name,
			namespace,
		)
	}
	return &app, nil
}

func IsApplicationHealthyAndSynced(
	app *argocd.Application,
	revision string,
) (bool, string) {
	if app.Status.Health.Status != health.HealthStatusHealthy {
		return false, fmt.Sprintf(
			"Argo CD Application %q in namespace %q has health state %q",
			app.Name,
			app.Namespace,
			app.Status.Health.Status,
		)
	}

	if app.Status.Sync.Status != argocd.SyncStatusCodeSynced ||
		(revision != "" && app.Status.Sync.Revision != revision) {
		return false, fmt.Sprintf(
			"Argo CD Application %q in namespace %q is not synced to revision %q",
			app.Name,
			app.Namespace,
			revision,
		)
	}

	return true, ""
}
