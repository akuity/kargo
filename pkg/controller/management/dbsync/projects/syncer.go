package projects

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
)

type projectStore interface {
	DeleteReplacedProject(context.Context, database.DeleteReplacedProjectParams) error
	UpsertProject(context.Context, database.UpsertProjectParams) error
	DeleteProjectByName(context.Context, string) error
	ListProjects(context.Context) ([]database.ProjectRow, error)
}

type syncer struct {
	reader client.Reader
	store  projectStore
}

// NewSyncer creates a Project syncer. Diff lists Projects through reader,
// which may be the manager's cached client; see syncapi.Syncer.
func NewSyncer(reader client.Reader, store projectStore) syncapi.Syncer {
	return &syncer{reader: reader, store: store}
}

func (*syncer) NewObject() client.Object { return &kargoapi.Project{} }

// Sources watches Projects alone. A Project row records nothing owned by
// another kind, so no other change can make it stale.
func (*syncer) Sources(c cache.Cache) []reconciler.Source[reconcile.Request] {
	return []reconciler.Source[reconcile.Request]{kube.Objects(c, &kargoapi.Project{})}
}

func (s *syncer) Sync(ctx context.Context, obj client.Object) error {
	project, ok := obj.(*kargoapi.Project)
	if !ok {
		return fmt.Errorf("expected a Project, got %T", obj)
	}
	params, err := toUpsertParams(project)
	if err != nil {
		return err
	}
	// A Project recreated under the same name leaves its predecessor's row
	// behind if its deletion was never seen. Deleting that row first takes
	// everything that belonged to the old Project with it. The two writes
	// need no transaction: if the upsert fails, the key is retried and the
	// row inserted, and the old Project is gone either way.
	if err = s.store.DeleteReplacedProject(ctx, database.DeleteReplacedProjectParams{
		Name: params.Name,
		ID:   params.ID,
	}); err != nil {
		return fmt.Errorf("error deleting replaced project: %w", err)
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
