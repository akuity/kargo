package health

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// FakeEngine is a mock implementation of the Engine interface that can be used
// to facilitate unit testing.
type FakeEngine struct {
	CheckFn func(ctx context.Context, project, stage string, criteria []Criteria) kargoapi.Health
}

// Check implements the Engine interface.
func (e *FakeEngine) Check(
	ctx context.Context,
	project string,
	stage string,
	criteria []Criteria,
) kargoapi.Health {
	if e.CheckFn == nil {
		return kargoapi.Health{Status: kargoapi.HealthStateHealthy}
	}
	return e.CheckFn(ctx, project, stage, criteria)
}
