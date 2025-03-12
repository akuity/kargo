package directives

import "context"

// mockPromoter is a mock implementation of the Promoter interface, which can be
// used for testing.
type mockPromoter struct {
	// name is the name of the Promoter.
	name string
	// promoteFunc is the function that the step should call when Promote is
	// called. If set, this function will be called instead of returning
	// promoteResult and promoteErr.
	promoteFunc func(context.Context, *PromotionStepContext) (PromotionStepResult, error)
	// promoteResult is the result that the Promoter should return when
	// Promote is called.
	promoteResult PromotionStepResult
	// promoteErr is the error that the Promoter should return when Promote is
	// called.
	promoteErr error
}

// Name implements the NamedRunner interface.
func (m *mockPromoter) Name() string {
	return m.name
}

// Promote implements the Promoter interface.
func (m *mockPromoter) Promote(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if m.promoteFunc != nil {
		return m.promoteFunc(ctx, stepCtx)
	}
	return m.promoteResult, m.promoteErr
}
