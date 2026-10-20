package stages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	sameSelectors, err := equalSelectors(row.TargetSelectors, params.TargetSelectors)
	if err != nil {
		return false, err
	}
	// Compare source fields at PostgreSQL's precision, excluding synced_at and
	// the rollout counter, which the mirror does not write.
	return row.ProjectName == params.ProjectName &&
		row.Name == params.Name &&
		sameSelectors &&
		row.CreatedAt.Equal(params.CreatedAt.Truncate(time.Microsecond)), nil
}

// equalSelectors compares stored selectors with encoded ones. PostgreSQL
// stores JSONB in a normal form of its own, so the stored ones are decoded
// and encoded again, as the encoded ones were, before the bytes are compared.
// NULL equals only nil.
func equalSelectors(stored json.RawMessage, encoded []byte) (bool, error) {
	if stored == nil || encoded == nil {
		return stored == nil && encoded == nil, nil
	}
	var selectors []metav1.LabelSelector
	if err := json.Unmarshal(stored, &selectors); err != nil {
		return false, fmt.Errorf("error decoding mirrored target selectors: %w", err)
	}
	normal, err := json.Marshal(selectors)
	if err != nil {
		return false, fmt.Errorf("error encoding mirrored target selectors: %w", err)
	}
	return bytes.Equal(normal, encoded), nil
}

func rowKey(row database.StageRow) client.ObjectKey {
	return client.ObjectKey{Namespace: row.ProjectName, Name: row.Name}
}
