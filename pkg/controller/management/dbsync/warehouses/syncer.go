package warehouses

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
)

type warehouseStore interface {
	UpsertWarehouse(context.Context, database.UpsertWarehouseParams) error
	DeleteWarehouseByName(context.Context, database.DeleteWarehouseByNameParams) error
	ListWarehouses(context.Context) ([]database.Warehouse, error)
	DeleteWarehousesByID(context.Context, []string) error
}

type syncer struct {
	reader client.Reader
	store  warehouseStore
}

// NewSyncer creates a Warehouse syncer using an uncached Kubernetes reader.
func NewSyncer(reader client.Reader, store warehouseStore) syncapi.Syncer {
	return &syncer{reader: reader, store: store}
}

func (*syncer) NewObject() client.Object { return &kargoapi.Warehouse{} }

func (s *syncer) Sync(ctx context.Context, obj client.Object) error {
	warehouse, ok := obj.(*kargoapi.Warehouse)
	if !ok {
		return fmt.Errorf("expected a Warehouse, got %T", obj)
	}
	project := &kargoapi.Project{}
	if err := s.reader.Get(ctx, client.ObjectKey{Name: warehouse.Namespace}, project); err != nil {
		// A missing parent is a retryable mapping failure, not a Warehouse deletion.
		return fmt.Errorf("error reading warehouse's project: %w", err)
	}
	params, err := toUpsertParams(warehouse, project)
	if err != nil {
		return err
	}
	if err = s.store.UpsertWarehouse(ctx, params); err != nil {
		return fmt.Errorf("error syncing warehouse: %w", err)
	}
	return nil
}

func (s *syncer) Delete(ctx context.Context, key client.ObjectKey) error {
	if err := s.store.DeleteWarehouseByName(ctx, database.DeleteWarehouseByNameParams{
		ProjectName: key.Namespace,
		Name:        key.Name,
	}); err != nil {
		return fmt.Errorf("error marking warehouse deleted: %w", err)
	}
	return nil
}

func (s *syncer) DeleteByIDs(ctx context.Context, ids []string) error {
	if err := s.store.DeleteWarehousesByID(ctx, ids); err != nil {
		return fmt.Errorf("error pruning warehouse rows: %w", err)
	}
	return nil
}

func toUpsertParams(warehouse *kargoapi.Warehouse, project *kargoapi.Project) (database.UpsertWarehouseParams, error) {
	if warehouse.UID == "" || project.UID == "" || warehouse.CreationTimestamp.IsZero() {
		return database.UpsertWarehouseParams{}, fmt.Errorf(
			"warehouse %q or its project has incomplete identity metadata", client.ObjectKeyFromObject(warehouse),
		)
	}
	return database.UpsertWarehouseParams{
		ID:        string(warehouse.UID),
		ProjectID: string(project.UID),
		Name:      warehouse.Name,
		CreatedAt: warehouse.CreationTimestamp.UTC(),
	}, nil
}
