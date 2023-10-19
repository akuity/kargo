package argocd

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
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
