package stages

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
)

type stageStore interface {
	UpsertStage(context.Context, database.UpsertStageParams) error
	DeleteStageByName(context.Context, database.DeleteStageByNameParams) error
	ListStages(context.Context) ([]database.Stage, error)
	DeleteStagesByID(context.Context, []string) error
}

type syncer struct {
	reader client.Reader
	store  stageStore
}

// NewSyncer creates a Stage syncer using an uncached Kubernetes reader.
func NewSyncer(reader client.Reader, store stageStore) syncapi.Syncer {
	return &syncer{reader: reader, store: store}
}

func (*syncer) NewObject() client.Object { return &kargoapi.Stage{} }

func (s *syncer) Sync(ctx context.Context, obj client.Object) error {
	stage, ok := obj.(*kargoapi.Stage)
	if !ok {
		return fmt.Errorf("expected a Stage, got %T", obj)
	}
	project := &kargoapi.Project{}
	if err := s.reader.Get(ctx, client.ObjectKey{Name: stage.Namespace}, project); err != nil {
		// A missing parent is a retryable mapping failure, not a Stage deletion.
		return fmt.Errorf("error reading stage's project: %w", err)
	}
	params, err := toUpsertParams(stage, project)
	if err != nil {
		return err
	}
	if err = s.store.UpsertStage(ctx, params); err != nil {
		return fmt.Errorf("error syncing stage: %w", err)
	}
	return nil
}

func (s *syncer) Delete(ctx context.Context, key client.ObjectKey) error {
	if err := s.store.DeleteStageByName(ctx, database.DeleteStageByNameParams{
		ProjectName: key.Namespace,
		Name:        key.Name,
	}); err != nil {
		return fmt.Errorf("error deleting stage row: %w", err)
	}
	return nil
}

func (s *syncer) DeleteByIDs(ctx context.Context, ids []string) error {
	if err := s.store.DeleteStagesByID(ctx, ids); err != nil {
		return fmt.Errorf("error pruning stage rows: %w", err)
	}
	return nil
}

func toUpsertParams(stage *kargoapi.Stage, project *kargoapi.Project) (database.UpsertStageParams, error) {
	if stage.UID == "" || project.UID == "" || stage.CreationTimestamp.IsZero() {
		return database.UpsertStageParams{}, fmt.Errorf(
			"stage %q or its project has incomplete identity metadata", client.ObjectKeyFromObject(stage),
		)
	}
	return database.UpsertStageParams{
		ID:        string(stage.UID),
		ProjectID: string(project.UID),
		Name:      stage.Name,
		CreatedAt: stage.CreationTimestamp.UTC(),
	}, nil
}
