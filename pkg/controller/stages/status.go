package stages

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/kubeclient"
)

// patchStageStatus patches the Stage's status to newStatus, leaving
// status.metadata exactly as it was last persisted. Metadata is written by
// set-metadata promotion steps, possibly in the middle of a reconcile pass, and
// this reconciler never changes it. Taking it from stage, the latest persisted
// state, keeps it out of the diff, so a pass that started from an older
// snapshot cannot revert or delete a concurrent write.
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
