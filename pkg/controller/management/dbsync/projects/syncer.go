package projects

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	GetProjectByName(context.Context, string) (database.ProjectRow, error)
}

// configStore keeps a Project's ProjectConfig row in step with the Project.
// A database row does not disappear with the Project's namespace the way the
// old custom resource did, so the syncer deletes it when it sees the Project
// deleted, and otherwise rebinds it to a Project of the same name with a new
// UID.
type configStore interface {
	AdoptForProject(ctx context.Context, project, newUID string) (bool, error)
	DeleteForProject(ctx context.Context, project, projectUID string) error
}

type syncer struct {
	reader  client.Reader
	store   projectStore
	configs configStore
}

// NewSyncer creates a Project syncer. Diff lists Projects through reader,
// which may be the manager's cached client; see syncapi.Syncer.
func NewSyncer(
	reader client.Reader,
	store projectStore,
	configs configStore,
) syncapi.Syncer {
	return &syncer{reader: reader, store: store, configs: configs}
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
	// A Project of the same name with a new UID leaves its predecessor's row
	// behind if its deletion was never seen. Deleting that row takes
	// everything that belonged to the old Project with it, except for the
	// ProjectConfig row, which is adopted by the new Project first instead:
	// a cluster restored from a backup gives every Project a new UID, and
	// the ProjectConfig lives only in the database. The writes need no
	// transaction: each is its own statement, and if any fails the key is
	// retried.
	if _, err = s.configs.AdoptForProject(ctx, params.Name, params.ID); err != nil {
		return fmt.Errorf("error adopting project config: %w", err)
	}
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
	// The mirrored row holds the UID of the Project being deleted, which is
	// what identifies its ProjectConfig row. With no mirror there is no
	// config to delete. If the config delete fails, the project row stays so
	// the retried key can find the UID again.
	row, err := s.store.GetProjectByName(ctx, key.Name)
	switch {
	case err == nil:
		if err = s.configs.DeleteForProject(ctx, key.Name, row.ID); err != nil {
			return fmt.Errorf("error deleting project config: %w", err)
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("error getting project row: %w", err)
	}
	if err = s.store.DeleteProjectByName(ctx, key.Name); err != nil {
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
