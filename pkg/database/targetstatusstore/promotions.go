package targetstatusstore

import (
	"context"
	"fmt"
	"time"

	"github.com/akuity/kargo/pkg/database"
)

// PromotionCursor marks a place in a Target's Promotions, which are ordered
// by creation time and then id: the Promotion a page continues before.
type PromotionCursor struct {
	CreatedAt time.Time
	ID        string
}

// PromotionListOptions narrows and bounds a list of a Target's Promotions.
type PromotionListOptions struct {
	// Stage, if given, narrows the list to the Stage's Promotions.
	Stage string
	// Before, if given, lists the Promotions older than the one at the
	// cursor.
	Before *PromotionCursor
	// Limit is the page size; DefaultPageSize if not positive.
	Limit int32
}

// ListPromotions returns one page of the Promotions to the named Target in
// the Project, newest first, as the options narrow and bound it. They are
// the mirror's records, which outlive the Promotions themselves. A page
// shorter than the limit is the last. A Target with no Promotions, or none
// at all, yields an empty page.
func (s *store) ListPromotions(
	ctx context.Context,
	project string,
	target string,
	opts PromotionListOptions,
) ([]Promotion, error) {
	args := database.ListTargetPromotionsParams{
		ProjectName: project,
		TargetName:  target,
		PageSize:    opts.Limit,
	}
	if args.PageSize <= 0 {
		args.PageSize = DefaultPageSize
	}
	if opts.Stage != "" {
		args.StageName = &opts.Stage
	}
	if opts.Before != nil {
		args.BeforeCreatedAt = &opts.Before.CreatedAt
		args.BeforeID = &opts.Before.ID
	}
	rows, err := s.queries.ListTargetPromotions(ctx, args)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing Promotions to Target %q in Project %q: %w", target, project, err,
		)
	}
	promotions := make([]Promotion, len(rows))
	for i, row := range rows {
		promotions[i] = Promotion{
			ID:         row.ID,
			Name:       row.Name,
			Stage:      row.StageName,
			Freight:    row.FreightName,
			Phase:      row.Phase,
			Rollout:    row.RolloutNumber,
			CreatedAt:  row.CreatedAt,
			FinishedAt: row.FinishedAt,
		}
	}
	return promotions, nil
}
