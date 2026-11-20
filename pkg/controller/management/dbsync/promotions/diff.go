package promotions

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
)

// Diff differs from other kinds' in what it snapshots. The table holds
// every Promotion there has ever been and is never pruned, so listing it
// whole each interval would cost ever more; and since no row is ever
// deleted, a finished row without an object is of no interest. Diff
// therefore snapshots the rows of the Promotions Kubernetes has now, which
// are bounded in number, and the unfinished rows, which are few: one of
// those without an object belongs to a Promotion deleted before it finished,
// whose deletion the mirror may have missed, and is queued so that Delete
// records it as finished. Reading Kubernetes before the snapshot is safe: a
// Promotion created between the two is listed as missing its row and
// re-synced, and one deleted between them is simply not listed.
func (s *syncer) Diff(ctx context.Context) ([]reconcile.Request, error) {
	promotions := &kargoapi.PromotionList{}
	if err := s.reader.List(ctx, promotions); err != nil {
		return nil, fmt.Errorf("error listing promotions for database resync: %w", err)
	}
	keys := database.ListPromotionsByKeyParams{
		ProjectNames: make([]string, 0, len(promotions.Items)),
		Names:        make([]string, 0, len(promotions.Items)),
	}
	for i := range promotions.Items {
		keys.ProjectNames = append(keys.ProjectNames, promotions.Items[i].Namespace)
		keys.Names = append(keys.Names, promotions.Items[i].Name)
	}
	rows, err := s.store.ListPromotionsByKey(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("error listing mirrored promotions: %w", err)
	}
	unfinished, err := s.store.ListUnfinishedPromotions(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing unfinished mirrored promotions: %w", err)
	}
	snapshot := make(map[client.ObjectKey]database.PromotionRow, len(rows)+len(unfinished))
	for _, row := range rows {
		snapshot[rowKey(row)] = row
	}
	for _, row := range unfinished {
		snapshot[rowKey(row)] = row
	}
	return syncapi.Diff(
		ctx,
		s.reader,
		&kargoapi.PromotionList{},
		client.ObjectKeyFromObject,
		snapshot,
		matches,
		rowKey,
	)
}

func matches(promotion *kargoapi.Promotion, row database.PromotionRow) (bool, error) {
	params, err := toUpsertParams(promotion)
	if err != nil {
		return false, err
	}
	// Compare source fields at PostgreSQL's precision, excluding synced_at.
	// Project and name are the row's key, so they match already.
	return row.StageName == params.StageName &&
		row.FreightName == params.FreightName &&
		equalStrings(row.TargetName, params.TargetName) &&
		row.Phase == params.Phase &&
		row.CreatedAt.Equal(params.CreatedAt) &&
		equalTimes(row.FinishedAt, params.FinishedAt) &&
		linksRollout(row, params), nil
}

// linksRollout reports whether the row records the PromotionRollout the
// Promotion names, as far as the upsert would change it. A recorded link is
// kept whatever the Promotion later claims, so only a missing one differs:
// one the Promotion's spec names but the row lacks, for instance because
// the rollout was not yet stored when the mirror first saw the Promotion. An id naming no rollout
// of the Promotion's Project is never recorded, so its row is queued again
// each resync; the upsert that follows writes nothing.
func linksRollout(row database.PromotionRow, params database.UpsertPromotionParams) bool {
	return row.PromotionRolloutID != nil || params.PromotionRolloutID == nil
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

func rowKey(row database.PromotionRow) client.ObjectKey {
	return client.ObjectKey{Namespace: row.ProjectName, Name: row.Name}
}
