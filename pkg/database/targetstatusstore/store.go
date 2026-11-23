// Package targetstatusstore reads what the database has on record about
// promotion to a Project's Targets, for the views the API server serves of
// them. Nothing in the database records a Target's status: everything here
// is read from the mirrored Promotions and the PromotionRollouts, and this
// store only reads.
package targetstatusstore

import (
	"context"
	"time"

	"github.com/akuity/kargo/pkg/database"
)

// DefaultPageSize is how many rows a list returns when it is given no limit.
const DefaultPageSize = 100

// Promotion is a Promotion to a Target, as the mirror recorded it.
type Promotion struct {
	// ID orders Promotions created at the same time.
	ID      string
	Name    string
	Stage   string
	Freight string
	Phase   string
	// Rollout is the number, among its Stage's, of the PromotionRollout that
	// created the Promotion, or zero if it was created on its own.
	Rollout    int64
	CreatedAt  time.Time
	FinishedAt *time.Time
}

// Store reads what is on record about promotion to Targets.
type Store interface {
	Targets(context.Context, string) ([]database.ListTargetLabelsRow, error)
	List(context.Context, Scope, []State, int32) ([]Entry, error)
	Summarize(context.Context, Scope) (Summary, error)
}

type store struct {
	queries *database.Queries
}

// New returns a Store that runs its statements through queries.
func New(queries *database.Queries) Store {
	return &store{queries: queries}
}
