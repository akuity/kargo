package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
)

// getStage is a helper to get a stage by namespace/name
func getStage(ctx context.Context, kc kubernetes.Client, project string, name string) (*kargoapi.Stage, error) {
	var stage kargoapi.Stage
	objKey := client.ObjectKey{
		Namespace: project,
		Name:      name,
	}
	err := kc.Get(ctx, objKey, &stage)
	if err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("stage %q not found", name))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &stage, nil
}
