package external

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

type targetResult struct {
	Kind           kargoapi.GenericWebhookTargetKind `json:"kind"`
	ListError      error                             `json:"listError,omitempty"`
	RefreshResults []refreshResult                   `json:"refreshResults,omitempty"`
}

type refreshResult struct {
	Success string `json:"success,omitempty"`
	Failure string `json:"failure,omitempty"`
}

func handleAction(
	ctx context.Context,
	c client.Client,
	project string,
	actionEnv map[string]any,
	action kargoapi.GenericWebhookAction,
) []targetResult {
	logger := logging.LoggerFromContext(ctx)
	targetResults := make([]targetResult, len(action.Targets))
	switch action.Name {
	case kargoapi.GenericWebhookActionNameRefresh:
		for i, target := range action.Targets {
			tLogger := logger.WithValues("targetKind", target.Kind)
			targetResults[i] = targetResult{Kind: target.Kind}
			objects, err := listTargetObjects(ctx, c, project, target, actionEnv)
			if err != nil {
				tLogger.Error(err, "failed to list objects for target")
				targetResults[i].ListError = fmt.Errorf("failed to list target objects: %w", err)
				continue
			}
			targetResults[i].RefreshResults = refreshObjects(ctx, c, target.Name, objects)
		}
	}
	return targetResults
}
