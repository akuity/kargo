package health

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/health"
)

// MockAggregatingChecker is a mock implementation of the AggregatingChecker
// interface that can be used to facilitate unit testing.
type MockAggregatingChecker struct {
	CheckFn func(ctx context.Context, project, stage string, criteria []health.Criteria) kargoapi.Health
}

// Check implements the AggregatingChecker interface.
func (m *MockAggregatingChecker) Check(
	ctx context.Context,
	project string,
	stage string,
	criteria []health.Criteria,
) kargoapi.Health {
	if m.CheckFn == nil {
		return kargoapi.Health{Status: kargoapi.HealthStateHealthy}
	}
	return m.CheckFn(ctx, project, stage, criteria)
}
