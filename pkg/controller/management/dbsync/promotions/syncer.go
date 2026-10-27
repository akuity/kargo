// Package promotions mirrors Promotions into the database as a history:
// rows are written and refreshed while a Promotion exists in Kubernetes and
// kept once it is gone. A Promotion deleted before it finished is recorded
// as Aborted, so that no row claims a Promotion is running that no longer
// exists.
package promotions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
)

type promotionStore interface {
	UpsertPromotion(context.Context, database.UpsertPromotionParams) (bool, error)
	ListPromotionsByKey(context.Context, database.ListPromotionsByKeyParams) ([]database.PromotionRow, error)
	ListUnfinishedPromotions(context.Context) ([]database.PromotionRow, error)
	FinishDeletedPromotion(context.Context, database.FinishDeletedPromotionParams) (int64, error)
}

type syncer struct {
	reader client.Reader
	store  promotionStore
}

// NewSyncer creates a Promotion syncer. Diff lists Promotions through reader,
// which may be the manager's cached client; see syncapi.Syncer.
func NewSyncer(reader client.Reader, store promotionStore) syncapi.Syncer {
	return &syncer{reader: reader, store: store}
}

func (*syncer) NewObject() client.Object { return &kargoapi.Promotion{} }

// Sources watches Promotions alone. A Promotion row records its Project,
// Stage, and Freight by name, which no change to any of them can alter, so
// no other change can make it stale.
func (*syncer) Sources(c cache.Cache) []reconciler.Source[reconcile.Request] {
	return []reconciler.Source[reconcile.Request]{kube.Objects(c, &kargoapi.Promotion{})}
}

func (s *syncer) Sync(ctx context.Context, obj client.Object) error {
	promotion, ok := obj.(*kargoapi.Promotion)
	if !ok {
		return fmt.Errorf("expected a Promotion, got %T", obj)
	}
	params, err := toUpsertParams(promotion)
	if err != nil {
		return err
	}
	// The id is used only if this writes the Promotion's row for the first
	// time. It is minted from the Promotion's creation time, so that ids sort
	// in creation order even for Promotions first seen long after.
	params.ID = ulid.MustNew(ulid.Timestamp(params.CreatedAt), ulid.DefaultEntropy()).String()
	nameTaken, err := s.store.UpsertPromotion(ctx, params)
	if err != nil {
		return fmt.Errorf("error syncing promotion: %w", err)
	}
	// Kargo never gives a name twice, so this takes a Promotion made some
	// other way. The record already under the name is kept; failing makes
	// the conflict visible and retries it, in case the record is the one
	// that is wrong and gets corrected.
	if nameTaken {
		return fmt.Errorf(
			"promotion %q has the name of an earlier Promotion, whose record was kept",
			client.ObjectKeyFromObject(promotion),
		)
	}
	return nil
}

// Delete keeps the row. A Promotion deleted from Kubernetes, by the garbage
// collector or anyone else, stays in the database as the record of what it
// did. One deleted before it finished, which Kubernetes allows since a
// Promotion has no finalizer, is recorded as Aborted as of now: its row
// would otherwise say it is running for good.
func (s *syncer) Delete(ctx context.Context, key client.ObjectKey) error {
	if _, err := s.store.FinishDeletedPromotion(ctx, database.FinishDeletedPromotionParams{
		Phase:       string(kargoapi.PromotionPhaseAborted),
		ProjectName: key.Namespace,
		Name:        key.Name,
	}); err != nil {
		return fmt.Errorf("error recording deleted promotion as finished: %w", err)
	}
	return nil
}

func toUpsertParams(promotion *kargoapi.Promotion) (database.UpsertPromotionParams, error) {
	if promotion.Namespace == "" || promotion.Name == "" || promotion.CreationTimestamp.IsZero() {
		return database.UpsertPromotionParams{}, fmt.Errorf(
			"promotion %q has incomplete identity metadata", client.ObjectKeyFromObject(promotion),
		)
	}
	params := database.UpsertPromotionParams{
		ProjectName: promotion.Namespace,
		Name:        promotion.Name,
		StageName:   promotion.Spec.Stage,
		FreightName: promotion.Spec.Freight,
		Phase:       string(promotion.Status.Phase),
		// At PostgreSQL's precision, since the upsert compares it with the
		// stored creation time to tell one Promotion from another.
		CreatedAt: promotion.CreationTimestamp.UTC().Truncate(time.Microsecond),
	}
	if target := promotion.Spec.Target; target != "" {
		params.TargetName = &target
	}
	if finished := promotion.Status.FinishedAt; finished != nil {
		finishedAt := finished.UTC()
		params.FinishedAt = &finishedAt
	}
	setFreight(promotion, &params)
	setPromotionRollout(promotion, &params)
	return params, nil
}

// setPromotionRollout carries the id of the PromotionRollout the Promotion
// was created by, if it names one, onto the upsert. The annotation is all
// that links the two; the rollout's creator does not record the link. An
// annotation that is not an id is recorded as no rollout rather than
// failing the sync, as is one naming no rollout of the Promotion's Project,
// which the upsert checks.
func setPromotionRollout(
	promotion *kargoapi.Promotion,
	params *database.UpsertPromotionParams,
) {
	value := promotion.Annotations[kargoapi.AnnotationKeyPromotionRollout]
	if value == "" {
		return
	}
	if id, err := uuid.Parse(value); err == nil {
		params.PromotionRolloutID = &id
	}
}

// setFreight carries the Promotion's FreightCollection, if it has one yet,
// onto the upsert as one origin and Freight name per entry, ordered by origin.
// Only names are recorded: what a piece of Freight holds is its own to say.
func setFreight(promotion *kargoapi.Promotion, params *database.UpsertPromotionParams) {
	refs := promotion.Status.FreightCollection.References()
	for _, ref := range refs {
		params.OriginKinds = append(params.OriginKinds, string(ref.Origin.Kind))
		params.OriginNames = append(params.OriginNames, ref.Origin.Name)
		params.FreightNames = append(params.FreightNames, ref.Name)
	}
}
