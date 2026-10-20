package promotions

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
)

// Diff differs from other kinds' in what it snapshots. The table holds
// every Promotion there has ever been and is never pruned, so listing it
// whole each interval would cost ever more; and since no row is ever
// deleted, a row without an object is of no interest. Diff therefore
// snapshots only the rows of the Promotions Kubernetes has now, which are
// bounded in number, and asks syncapi.Diff to leave the rest alone. Reading
// Kubernetes before the snapshot is safe for the same reason: a Promotion
// created between the two is listed as missing its row and re-synced, and
// one deleted between them is simply not listed.
func (s *syncer) Diff(ctx context.Context) ([]reconcile.Request, error) {
	promotions := &kargoapi.PromotionList{}
	if err := s.reader.List(ctx, promotions); err != nil {
		return nil, fmt.Errorf("error listing promotions for database resync: %w", err)
	}
	ids := make([]string, 0, len(promotions.Items))
	for i := range promotions.Items {
		ids = append(ids, string(promotions.Items[i].UID))
	}
	rows, err := s.store.ListPromotionsByID(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("error listing mirrored promotions: %w", err)
	}
	snapshot := make(map[string]database.PromotionRow, len(rows))
	for _, row := range rows {
		snapshot[row.ID] = row
	}
	return syncapi.Diff(ctx, s.reader, &kargoapi.PromotionList{}, snapshot, matches, nil)
}

func matches(promotion *kargoapi.Promotion, row database.PromotionRow) (bool, error) {
	params, err := toUpsertParams(promotion)
	if err != nil {
		return false, err
	}
	// Compare source fields at PostgreSQL's precision, excluding synced_at.
	return row.ProjectName == params.ProjectName &&
		row.Name == params.Name &&
		row.StageName == params.StageName &&
		row.FreightName == params.FreightName &&
		equalStrings(row.TargetName, params.TargetName) &&
		row.Phase == params.Phase &&
		row.CreatedAt.Equal(params.CreatedAt.Truncate(time.Microsecond)) &&
		equalTimes(row.FinishedAt, params.FinishedAt), nil
}

func equalStrings(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalTimes(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(b.Truncate(time.Microsecond))
}
