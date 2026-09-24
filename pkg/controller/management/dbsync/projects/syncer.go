package projects

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
)

type projectStore interface {
	UpsertProject(context.Context, database.UpsertProjectParams) error
	DeleteProjectByName(context.Context, string) error
	ListProjects(context.Context) ([]database.Project, error)
	DeleteProjectsByID(context.Context, []string) error
}

type syncer struct {
	reader client.Reader
	store  projectStore
}

// NewSyncer creates a Project syncer. The reader must bypass the Kubernetes cache.
func NewSyncer(reader client.Reader, store projectStore) syncapi.Syncer {
	return &syncer{reader: reader, store: store}
}

func (*syncer) NewObject() client.Object { return &kargoapi.Project{} }

func (s *syncer) Sync(ctx context.Context, obj client.Object) error {
	project, ok := obj.(*kargoapi.Project)
	if !ok {
		return fmt.Errorf("expected a Project, got %T", obj)
	}
	params, err := toUpsertParams(project)
	if err != nil {
		return err
	}
	if err = s.store.UpsertProject(ctx, params); err != nil {
		return fmt.Errorf("error syncing project: %w", err)
	}
	return nil
}

func (s *syncer) Delete(ctx context.Context, key client.ObjectKey) error {
	if err := s.store.DeleteProjectByName(ctx, key.Name); err != nil {
		return fmt.Errorf("error deleting project row: %w", err)
	}
	return nil
}

func (s *syncer) DeleteByIDs(ctx context.Context, ids []string) error {
	if err := s.store.DeleteProjectsByID(ctx, ids); err != nil {
		return fmt.Errorf("error pruning project rows: %w", err)
	}
	return nil
}

func toUpsertParams(project *kargoapi.Project) (database.UpsertProjectParams, error) {
	if project.UID == "" || project.CreationTimestamp.IsZero() {
		return database.UpsertProjectParams{}, fmt.Errorf("project %q has incomplete identity metadata", project.Name)
	}
	return database.UpsertProjectParams{
		ID:        string(project.UID),
		Name:      project.Name,
		CreatedAt: project.CreationTimestamp.UTC(),
	}, nil
}
