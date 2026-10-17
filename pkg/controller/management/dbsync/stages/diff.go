package stages

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

func (s *syncer) Diff(ctx context.Context) ([]reconcile.Request, error) {
	rows, err := s.store.ListStages(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing mirrored stages: %w", err)
	}
	snapshot := make(map[string]database.StageRow, len(rows))
	for _, row := range rows {
		snapshot[row.ID] = row
	}
	return syncapi.Diff(ctx, s.reader, &kargoapi.StageList{}, snapshot, matches, rowKey)
}

func matches(stage *kargoapi.Stage, row database.StageRow) (bool, error) {
	params, err := toUpsertParams(stage)
	if err != nil {
		return false, err
	}
	// Compare source metadata at PostgreSQL's precision, excluding synced_at.
	return row.ProjectName == params.ProjectName &&
		row.Name == params.Name &&
		row.CreatedAt.Equal(params.CreatedAt.Truncate(time.Microsecond)), nil
}

func rowKey(row database.StageRow) client.ObjectKey {
	return client.ObjectKey{Namespace: row.ProjectName, Name: row.Name}
}
