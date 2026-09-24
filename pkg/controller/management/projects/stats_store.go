package projects

import (
	"context"

	"github.com/akuity/kargo/pkg/database"
)

// projectStatsStore is the slice of database.Store the Project reconciler uses
// to compute a Project's Target stats.
type projectStatsStore interface {
	ListTargets(context.Context, string) ([]database.Target, error)
	ListPromotionRequests(context.Context, string) ([]database.PromotionRequestSnapshot, error)
}
