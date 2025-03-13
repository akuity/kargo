package health

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// FakeAggregatingChecker is a mock implementation of the AggregatingChecker
// interface that can be used to facilitate unit testing.
type FakeAggregatingChecker struct {
	CheckFn func(ctx context.Context, project, stage string, criteria []Criteria) kargoapi.Health
}

// Check implements the AggregatingChecker interface.
func (f *FakeAggregatingChecker) Check(
	ctx context.Context,
	project string,
	stage string,
	criteria []Criteria,
) kargoapi.Health {
	if f.CheckFn == nil {
		return kargoapi.Health{Status: kargoapi.HealthStateHealthy}
	}
	return f.CheckFn(ctx, project, stage, criteria)
}
