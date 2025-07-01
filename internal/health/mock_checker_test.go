package health

import (
	"context"

	"github.com/akuity/kargo/pkg/health"
)

// mockChecker is a mock implementation of the Checker interface, which can be
// used for testing.
type mockChecker struct {
	// name is the name of the Checker.
	name string
	// checkFunc is the function that this Checker should call when Check is
	// called. If set, this function will be called instead of returning
	// checkResult.
	checkFunc func(ctx context.Context, project, stage string, criteria health.Criteria) health.Result
	// checkResult is the result that this Checker should return when Check is
	// called.
	checkResult health.Result
}

// Name implements the Checker interface.
func (m *mockChecker) Name() string {
	return m.name
}

// Check implements the Checker interface.
func (m *mockChecker) Check(ctx context.Context, project, stage string, criteria health.Criteria) health.Result {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, project, stage, criteria)
	}
	return m.checkResult
}
