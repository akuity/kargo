package stages

import (
	"context"

	"github.com/akuity/kargo/pkg/database"
)

// promotionRequestStore is the slice of database.Store the Stage reconciler
// uses to create the PromotionRequests of a target-aware Stage and to follow
// their progress.
type promotionRequestStore interface {
	ListTargets(context.Context, string) ([]database.Target, error)
	ListPromotionRequestsByStage(
		context.Context,
		database.ListPromotionRequestsByStageParams,
	) ([]database.PromotionRequestSnapshot, error)
	GetPromotionRequest(
		context.Context,
		database.GetPromotionRequestParams,
	) (database.PromotionRequestSnapshot, error)
	PromotionRequestExists(context.Context, database.PromotionRequestExistsParams) (bool, error)
	CreatePromotionRequest(
		context.Context,
		database.PromotionRequestCreate,
	) (database.PromotionRequestSnapshot, error)
}
