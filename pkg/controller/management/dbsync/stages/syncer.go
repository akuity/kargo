package stages

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
)

type stageStore interface {
	DeleteReplacedStage(context.Context, database.DeleteReplacedStageParams) error
	UpsertStage(context.Context, database.UpsertStageParams) error
	DeleteStageByName(context.Context, database.DeleteStageByNameParams) error
	ListStages(context.Context) ([]database.StageRow, error)
}

type syncer struct {
	reader client.Reader
	store  stageStore
}

// NewSyncer creates a Stage syncer. Diff lists Stages through reader, which
// may be the manager's cached client; see syncapi.Syncer.
func NewSyncer(reader client.Reader, store stageStore) syncapi.Syncer {
	return &syncer{reader: reader, store: store}
}

func (*syncer) NewObject() client.Object { return &kargoapi.Stage{} }

// Sources watches Stages alone. A Stage row records its Project by name,
// which no change to the Project can alter, and its own selectors, so no
// other change can make it stale.
func (*syncer) Sources(c cache.Cache) []reconciler.Source[reconcile.Request] {
	return []reconciler.Source[reconcile.Request]{kube.Objects(c, &kargoapi.Stage{})}
}

func (s *syncer) Sync(ctx context.Context, obj client.Object) error {
	stage, ok := obj.(*kargoapi.Stage)
	if !ok {
		return fmt.Errorf("expected a Stage, got %T", obj)
	}
	params, err := toUpsertParams(stage)
	if err != nil {
		return err
	}
	// A Stage recreated under the same name leaves its predecessor's row
	// behind if its deletion was never seen. Deleting that row first keeps
	// the upsert from colliding with it. The two writes need no transaction:
	// if the upsert fails, the key is retried and the row inserted, and the
	// old Stage is gone either way.
	if err = s.store.DeleteReplacedStage(ctx, database.DeleteReplacedStageParams{
		ProjectName: params.ProjectName,
		Name:        params.Name,
		ID:          params.ID,
	}); err != nil {
		return fmt.Errorf("error deleting replaced stage: %w", err)
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

func toUpsertParams(stage *kargoapi.Stage) (database.UpsertStageParams, error) {
	if stage.UID == "" || stage.Namespace == "" || stage.CreationTimestamp.IsZero() {
		return database.UpsertStageParams{}, fmt.Errorf(
			"stage %q has incomplete identity metadata", client.ObjectKeyFromObject(stage),
		)
	}
	selectors, err := targetSelectors(stage)
	if err != nil {
		return database.UpsertStageParams{}, err
	}
	return database.UpsertStageParams{
		ID:              string(stage.UID),
		ProjectName:     stage.Namespace,
		Name:            stage.Name,
		TargetSelectors: selectors,
		CreatedAt:       stage.CreationTimestamp.UTC(),
	}, nil
}

// targetSelectors encodes the selectors by which a Stage governs Targets. A
// classic Stage has none and yields nil, stored as NULL. A target-aware Stage
// yields a JSON list, empty if it lists no selectors, so the two stay apart.
func targetSelectors(stage *kargoapi.Stage) ([]byte, error) {
	if stage.Spec.Targets == nil {
		return nil, nil
	}
	selectors := stage.Spec.Targets.Selectors
	if selectors == nil {
		selectors = []metav1.LabelSelector{}
	}
	encoded, err := json.Marshal(selectors)
	if err != nil {
		return nil, fmt.Errorf(
			"error encoding target selectors of stage %q: %w",
			client.ObjectKeyFromObject(stage), err,
		)
	}
	return encoded, nil
}
