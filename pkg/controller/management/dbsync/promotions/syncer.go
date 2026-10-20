// Package promotions mirrors Promotions into the database as a history:
// rows are written and refreshed while a Promotion exists in Kubernetes and
// kept once it is gone.
package promotions

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

type promotionStore interface {
	UpsertPromotion(context.Context, database.UpsertPromotionParams) error
	ListPromotionsByID(context.Context, []string) ([]database.PromotionRow, error)
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
	if err = s.store.UpsertPromotion(ctx, params); err != nil {
		return fmt.Errorf("error syncing promotion: %w", err)
	}
	return nil
}

// Delete keeps the row. A Promotion deleted from Kubernetes, by the garbage
// collector or anyone else, stays in the database as the record of what it
// did.
func (*syncer) Delete(context.Context, client.ObjectKey) error { return nil }

func toUpsertParams(promotion *kargoapi.Promotion) (database.UpsertPromotionParams, error) {
	if promotion.UID == "" || promotion.Namespace == "" || promotion.CreationTimestamp.IsZero() {
		return database.UpsertPromotionParams{}, fmt.Errorf(
			"promotion %q has incomplete identity metadata", client.ObjectKeyFromObject(promotion),
		)
	}
	params := database.UpsertPromotionParams{
		ID:          string(promotion.UID),
		ProjectName: promotion.Namespace,
		Name:        promotion.Name,
		StageName:   promotion.Spec.Stage,
		FreightName: promotion.Spec.Freight,
		Phase:       string(promotion.Status.Phase),
		CreatedAt:   promotion.CreationTimestamp.UTC(),
	}
	if target := promotion.Spec.Target; target != "" {
		params.TargetName = &target
	}
	if finished := promotion.Status.FinishedAt; finished != nil {
		finishedAt := finished.UTC()
		params.FinishedAt = &finishedAt
	}
	setFreight(promotion, &params)
	return params, nil
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
