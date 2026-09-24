package freight

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
)

type freightStore interface {
	UpsertFreight(context.Context, database.FreightUpsert) error
	GetFreightWarehouseID(context.Context, string) (string, error)
	DeleteFreightByName(context.Context, database.DeleteFreightByNameParams) error
	ListFreight(context.Context) ([]database.FreightSnapshot, error)
	DeleteFreightByIDs(context.Context, []string) error
}

type syncer struct {
	reader client.Reader
	store  freightStore
}

// NewSyncer creates a Freight syncer using an uncached Kubernetes reader.
func NewSyncer(reader client.Reader, store freightStore) syncapi.Syncer {
	return &syncer{reader: reader, store: store}
}

func (*syncer) NewObject() client.Object { return &kargoapi.Freight{} }

func (s *syncer) Sync(ctx context.Context, obj client.Object) error {
	freight, ok := obj.(*kargoapi.Freight)
	if !ok {
		return fmt.Errorf("expected Freight, got %T", obj)
	}
	project := &kargoapi.Project{}
	if err := s.reader.Get(ctx, client.ObjectKey{Name: freight.Namespace}, project); err != nil {
		return fmt.Errorf("error reading freight's project: %w", err)
	}
	warehouseID, err := s.store.GetFreightWarehouseID(ctx, string(freight.UID))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		warehouse := &kargoapi.Warehouse{}
		if err = s.reader.Get(ctx, client.ObjectKey{
			Namespace: freight.Namespace, Name: freight.Origin.Name,
		}, warehouse); err != nil {
			return fmt.Errorf("error reading freight's warehouse: %w", err)
		}
		warehouseID = string(warehouse.UID)
	case err != nil:
		return fmt.Errorf("error reading freight's stored warehouse identity: %w", err)
	}
	// Reuse an established association even if the Warehouse is now absent or
	// its name belongs to another UID. Freight origins identify names, not UIDs.
	params, err := toUpsertParams(freight, project, warehouseID)
	if err != nil {
		return err
	}
	if err = s.store.UpsertFreight(ctx, params); err != nil {
		return fmt.Errorf("error syncing freight: %w", err)
	}
	return nil
}

func (s *syncer) Delete(ctx context.Context, key client.ObjectKey) error {
	if err := s.store.DeleteFreightByName(ctx, database.DeleteFreightByNameParams{
		ProjectName: key.Namespace, Name: key.Name,
	}); err != nil {
		return fmt.Errorf("error marking freight deleted: %w", err)
	}
	return nil
}

func (s *syncer) DeleteByIDs(ctx context.Context, ids []string) error {
	if err := s.store.DeleteFreightByIDs(ctx, ids); err != nil {
		return fmt.Errorf("error pruning freight rows: %w", err)
	}
	return nil
}

func toUpsertParams(
	freight *kargoapi.Freight,
	project *kargoapi.Project,
	warehouseID string,
) (database.FreightUpsert, error) {
	if freight.UID == "" || project.UID == "" || warehouseID == "" || freight.CreationTimestamp.IsZero() {
		return database.FreightUpsert{}, fmt.Errorf(
			"freight %q or its parents have incomplete identity metadata", client.ObjectKeyFromObject(freight),
		)
	}
	if freight.Origin.Kind != kargoapi.FreightOriginKindWarehouse || freight.Origin.Name == "" {
		return database.FreightUpsert{}, fmt.Errorf("invalid freight origin %q", freight.Origin.String())
	}
	if freight.EffectiveDiscoveredAt().IsZero() {
		return database.FreightUpsert{}, fmt.Errorf("freight %q has no discovery timestamp", freight.Name)
	}
	artifacts, err := toContents(freight)
	if err != nil {
		return database.FreightUpsert{}, err
	}
	return database.FreightUpsert{
		UpsertFreightParams: database.UpsertFreightParams{
			ID: string(freight.UID), ProjectID: string(project.UID), WarehouseID: warehouseID,
			Name: freight.Name, Alias: freight.Alias,
			CreatedAt: freight.CreationTimestamp.UTC(), DiscoveredAt: freight.EffectiveDiscoveredAt().UTC(),
		},
		FreightContents: artifacts,
	}, nil
}
