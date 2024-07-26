package v1alpha1

import (
	"context"
	"fmt"

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
) (*Application, error) {
	app := Application{}
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
		return nil, fmt.Errorf(
			"error getting Argo CD Application %q in namespace %q: %w",
			name,
			namespace,
			err,
		)
	}
	return &app, nil
}

// GetAppProject returns a pointer to the Argo CD AppProject resource
// If no such resource is found, nil is returned instead.
func GetAppProject(
	ctx context.Context,
	ctrlRuntimeClient client.Client,
	name string,
) (*AppProject, error) {
	appProject := AppProject{}
	if err := ctrlRuntimeClient.Get(
		ctx,
		client.ObjectKey{
			Name: name,
		},
		&appProject,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Argo CD AppProject %q: %w",
			name,
			err,
		)
	}
	return &appProject, nil
}
