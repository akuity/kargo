package database

import (
	"context"
	"fmt"
)

func (s *store) UpsertWarehouse(ctx context.Context, params UpsertWarehouseParams) error {
	return s.transact(ctx, func(txCtx context.Context, queries *Queries) error {
		if err := queries.DeleteReplacedWarehouse(txCtx, DeleteReplacedWarehouseParams{
			ProjectID: params.ProjectID, Name: params.Name, ID: params.ID,
		}); err != nil {
			return fmt.Errorf("error retiring replaced warehouse: %w", err)
		}
		return queries.UpsertWarehouse(txCtx, params)
	})
}

func (s *store) UpsertFreight(ctx context.Context, params FreightUpsert) error {
	return s.transact(ctx, func(txCtx context.Context, queries *Queries) error {
		if err := queries.DeleteReplacedFreight(txCtx, DeleteReplacedFreightParams{
			ProjectID: params.ProjectID, Name: params.Name, ID: params.ID,
		}); err != nil {
			return fmt.Errorf("error retiring replaced freight: %w", err)
		}
		// Existing Freight keeps its original Project and Warehouse identities,
		// including when its Warehouse has since been deleted or recreated.
		if err := queries.UpsertFreight(txCtx, params.UpsertFreightParams); err != nil {
			return err
		}
		return replaceFreightContents(txCtx, queries, params.ID, params.FreightContents)
	})
}

func (s *store) GetFreightWarehouseID(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.GetFreightWarehouseID(ctx, id)
}

func (s *store) DeleteWarehouseByName(ctx context.Context, params DeleteWarehouseByNameParams) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.DeleteWarehouseByName(ctx, params)
}

func (s *store) DeleteFreightByName(ctx context.Context, params DeleteFreightByNameParams) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.DeleteFreightByName(ctx, params)
}

func (s *store) ListWarehouses(ctx context.Context) ([]Warehouse, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.ListWarehouses(ctx)
}

func (s *store) DeleteWarehousesByID(ctx context.Context, ids []string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.DeleteWarehousesByID(ctx, ids)
}

func (s *store) DeleteFreightByIDs(ctx context.Context, ids []string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.DeleteFreightByIDs(ctx, ids)
}
