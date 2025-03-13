package health

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Engine is an interface for executing a sequence of health checks.
type Engine interface {
	// Check executes the specified sequence of health checks and returns a
	// kargoapi.Health that aggregates their results.
	Check(ctx context.Context, project, stage string, criteria []Criteria) kargoapi.Health
}
