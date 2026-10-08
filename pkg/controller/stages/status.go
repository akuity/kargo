package stages

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/kubeclient"
)

// patchStageStatus patches the Stage's status to newStatus but never patches
// status.metadata, which set-metadata steps may write mid-pass. It copies
// metadata from stage into newStatus (in place) so the server's value always
// wins.
func patchStageStatus(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
	newStatus *kargoapi.StageStatus,
) error {
	newStatus.Metadata = stage.Status.DeepCopy().Metadata
	return kubeclient.PatchStatus(ctx, c, stage, func(status *kargoapi.StageStatus) {
		*status = *newStatus
	})
}
