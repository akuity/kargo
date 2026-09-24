package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store maintains the database mirror. Upserts replace obsolete identities
// transactionally; callers never observe a partially replaced resource.
type Store interface {
	UpsertWarehouse(context.Context, UpsertWarehouseParams) error
	UpsertFreight(context.Context, FreightUpsert) error
	GetFreightWarehouseID(context.Context, string) (string, error)
	DeleteWarehouseByName(context.Context, DeleteWarehouseByNameParams) error
	DeleteFreightByName(context.Context, DeleteFreightByNameParams) error
	// ListWarehouses and ListFreight return only rows not marked deleted.
	ListWarehouses(context.Context) ([]Warehouse, error)
	ListFreight(context.Context) ([]FreightSnapshot, error)
	// Warehouse and Freight deletion retains identities for existing references.
	DeleteWarehousesByID(context.Context, []string) error
	DeleteFreightByIDs(context.Context, []string) error
	UpsertProject(context.Context, UpsertProjectParams) error
	UpsertStage(context.Context, UpsertStageParams) error
	DeleteProjectByName(context.Context, string) error
	DeleteStageByName(context.Context, DeleteStageByNameParams) error
	ListProjects(context.Context) ([]Project, error)
	ListStages(context.Context) ([]Stage, error)
	DeleteProjectsByID(context.Context, []string) error
	DeleteStagesByID(context.Context, []string) error
}

type store struct {
	pool    *pgxpool.Pool
	queries *Queries
}

// NewStore returns a mirror backed by the shared pool. The caller owns the pool.
func NewStore(pool *pgxpool.Pool) Store {
	return &store{pool: pool, queries: New(pool)}
}

func (s *store) UpsertProject(ctx context.Context, params UpsertProjectParams) error {
	return s.transact(ctx, func(txCtx context.Context, queries *Queries) error {
		if err := queries.DeleteReplacedProject(txCtx, DeleteReplacedProjectParams{
			Name: params.Name,
			ID:   params.ID,
		}); err != nil {
			return fmt.Errorf("error replacing project: %w", err)
		}
		return queries.UpsertProject(txCtx, params)
	})
}

func (s *store) UpsertStage(ctx context.Context, params UpsertStageParams) error {
	return s.transact(ctx, func(txCtx context.Context, queries *Queries) error {
		if err := queries.DeleteReplacedStage(txCtx, DeleteReplacedStageParams{
			ProjectID: params.ProjectID,
			Name:      params.Name,
			ID:        params.ID,
		}); err != nil {
			return fmt.Errorf("error replacing stage: %w", err)
		}
		return queries.UpsertStage(txCtx, params)
	})
}

func (s *store) transact(ctx context.Context, write func(context.Context, *Queries) error) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return write(ctx, s.queries.WithTx(tx))
	})
}

func (s *store) DeleteProjectByName(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.DeleteProjectByName(ctx, name)
}

func (s *store) DeleteStageByName(ctx context.Context, params DeleteStageByNameParams) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.DeleteStageByName(ctx, params)
}

func (s *store) ListProjects(ctx context.Context) ([]Project, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.ListProjects(ctx)
}

func (s *store) ListStages(ctx context.Context) ([]Stage, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.ListStages(ctx)
}

func (s *store) DeleteProjectsByID(ctx context.Context, ids []string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.DeleteProjectsByID(ctx, ids)
}

func (s *store) DeleteStagesByID(ctx context.Context, ids []string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.DeleteStagesByID(ctx, ids)
}
