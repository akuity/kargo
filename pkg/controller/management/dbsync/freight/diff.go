package freight

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
	rows, err := s.store.ListFreight(ctx)
	if err != nil {
		return syncapi.Changes{}, fmt.Errorf("error listing mirrored freight: %w", err)
	}
	snapshot := make(map[string]database.FreightSnapshot, len(rows))
	for _, row := range rows {
		snapshot[row.ID] = row
	}
	projects := &kargoapi.ProjectList{}
	if err = s.reader.List(ctx, projects); err != nil {
		return syncapi.Changes{}, fmt.Errorf("error listing freight parents for database resync: %w", err)
	}
	if projects.Continue != "" {
		return syncapi.Changes{}, errors.New("incomplete freight parent list for database resync")
	}
	parents := make(map[string]*kargoapi.Project, len(projects.Items))
	for i := range projects.Items {
		project := &projects.Items[i]
		parents[project.Name] = project
	}
	return syncapi.Diff(
		ctx,
		s.reader,
		&kargoapi.FreightList{},
		snapshot,
		func(freight *kargoapi.Freight, row database.FreightSnapshot) (bool, error) {
			project := parents[freight.Namespace]
			if project == nil {
				return false, nil
			}
			params, conversionErr := toUpsertParams(freight, project, row.WarehouseID)
			if conversionErr != nil {
				return false, conversionErr
			}
			return row.Name == params.Name && row.ProjectID == params.ProjectID && row.Alias == params.Alias &&
				row.CreatedAt.Equal(params.CreatedAt.Truncate(time.Microsecond)) &&
				row.DiscoveredAt.Equal(params.DiscoveredAt.Truncate(time.Microsecond)) &&
				equalContents(row.FreightContents, params.FreightContents), nil
		},
	)
}
