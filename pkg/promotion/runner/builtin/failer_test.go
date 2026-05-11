package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_failer_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:             "valid empty config",
			config:           promotion.Config{},
			expectedProblems: nil,
		},
		{
			name:             "valid config with message",
			config:           promotion.Config{"message": "test message"},
			expectedProblems: nil,
		},
	}

	r := newFailer(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*failer)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_failer_run(t *testing.T) {
	tests := []struct {
		name          string
		cfg           builtin.FailConfig
		expectedError string
	}{
		{
			"without message",
			builtin.FailConfig{},
			"failed",
		},
		{
			"with message",
			builtin.FailConfig{Message: "test message"},
			"failed: test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFailer(promotion.StepRunnerCapabilities{})
			runner, ok := r.(*failer)
			require.True(t, ok)

			res, err := runner.run(tt.cfg)

			var termErr *promotion.TerminalError
			require.ErrorAs(t, err, &termErr)
			require.EqualError(t, err, tt.expectedError)
			require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
		})
	}
}
