package projects

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
	rows, err := s.store.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing mirrored projects: %w", err)
	}
	snapshot := make(map[string]database.ProjectRow, len(rows))
	for _, row := range rows {
		snapshot[row.ID] = row
	}
	return syncapi.Diff(ctx, s.reader, &kargoapi.ProjectList{}, snapshot, matches, rowKey)
}

func matches(project *kargoapi.Project, row database.ProjectRow) (bool, error) {
	params, err := toUpsertParams(project)
	if err != nil {
		return false, err
	}
	// Compare source metadata at PostgreSQL's precision, excluding synced_at.
	return row.Name == params.Name &&
		row.CreatedAt.Equal(params.CreatedAt.Truncate(time.Microsecond)), nil
}

func rowKey(row database.ProjectRow) client.ObjectKey {
	return client.ObjectKey{Name: row.Name}
}
