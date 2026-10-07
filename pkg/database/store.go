package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// operationTimeout bounds every store operation so that a blocked query or an
// unreachable database cannot stall a reconciliation indefinitely.
const operationTimeout = 5 * time.Second

// Store maintains the database mirror. Upserts replace obsolete identities
// transactionally; callers never observe a partially replaced resource.
type Store interface {
	UpsertProject(context.Context, UpsertProjectParams) error
	DeleteProjectByName(context.Context, string) error
	ListProjects(context.Context) ([]ProjectRow, error)
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

func (s *store) ListProjects(ctx context.Context) ([]ProjectRow, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.ListProjects(ctx)
}
