package health

import "context"

// mockChecker is a mock implementation of the Checker interface, which can be
// used for testing.
type mockChecker struct {
	// name is the name of the Checker.
	name string
	// checkFunc is the function that this Checker should call when Check is
	// called. If set, this function will be called instead of returning
	// checkResult.
	checkFunc func(context.Context, Criteria) Result
	// checkResult is the result that this Checker should return when Check is
	// called.
	checkResult Result
}

// Name implements the Checker interface.
func (m *mockChecker) Name() string {
	return m.name
}

// Check implements the Checker interface.
func (m *mockChecker) Check(ctx context.Context, criteria Criteria) Result {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, criteria)
	}
	return m.checkResult
}
