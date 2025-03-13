package health

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// FakeMultiChecker is a mock implementation of the MultiChecker interface that
// can be used to facilitate unit testing.
type FakeMultiChecker struct {
	CheckFn func(ctx context.Context, project, stage string, criteria []Criteria) kargoapi.Health
}

// Check implements the MultiChecker interface.
func (f *FakeMultiChecker) Check(
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
