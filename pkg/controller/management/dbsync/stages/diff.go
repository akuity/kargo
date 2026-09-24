package stages

import (
	"context"
	"errors"
	"fmt"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
)

func (s *syncer) Diff(ctx context.Context) (syncapi.Changes, error) {
	// Snapshot database rows before either Kubernetes list.
	rows, err := s.store.ListStages(ctx)
	if err != nil {
		return syncapi.Changes{}, fmt.Errorf("error listing mirrored stages: %w", err)
	}
	snapshot := make(map[string]database.Stage, len(rows))
	for _, row := range rows {
		snapshot[row.ID] = row
	}
	// Resolve parents in one uncached list, rather than one request per Stage.
	projects := &kargoapi.ProjectList{}
	if err = s.reader.List(ctx, projects); err != nil {
		return syncapi.Changes{}, fmt.Errorf("error listing stage parents for database resync: %w", err)
	}
	if projects.Continue != "" {
		return syncapi.Changes{}, errors.New("incomplete stage parent list for database resync")
	}
	parents := make(map[string]*kargoapi.Project, len(projects.Items))
	for i := range projects.Items {
		project := &projects.Items[i]
		parents[project.Name] = project
	}
	return syncapi.Diff(
		ctx,
		s.reader,
		&kargoapi.StageList{},
		snapshot,
		func(stage *kargoapi.Stage, row database.Stage) (bool, error) {
			project := parents[stage.Namespace]
			if project == nil {
				// Reconciliation retries the missing parent; the live Stage's row
				// is preserved and unrelated stale rows can still be cleaned up.
				return false, nil
			}
			params, conversionErr := toUpsertParams(stage, project)
			if conversionErr != nil {
				return false, conversionErr
			}
			return row.Name == params.Name && row.ProjectID == params.ProjectID &&
				row.CreatedAt.Equal(params.CreatedAt.Truncate(time.Microsecond)), nil
		},
	)
}
