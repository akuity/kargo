package promotion

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_stepRunnerRegistry(t *testing.T) {
	// Generic name-based registries are well-tested in the component package, but
	// stepRunnerRegistry decorates a generic registry with specific behavior we'd
	// like to validate.

	const testStepKindName = "test"

	testCases := []struct {
		name  string
		setup func(*testing.T) StepRunnerRegistry
	}{
		{
			name: "MustNewStepRunnerRegistry defaults error threshold",
			setup: func(*testing.T) StepRunnerRegistry {
				return MustNewStepRunnerRegistry(
					StepRunnerRegistration{Name: testStepKindName},
				)
			},
		},
		{
			name: "Register defaults error threshold",
			setup: func(*testing.T) StepRunnerRegistry {
				r := MustNewStepRunnerRegistry()
				err := r.Register(StepRunnerRegistration{Name: testStepKindName})
				require.NoError(t, err)
				return r
			},
		},
		{
			name: "MustRegister defaults error threshold",
			setup: func(*testing.T) StepRunnerRegistry {
				r := MustNewStepRunnerRegistry()
				r.MustRegister(StepRunnerRegistration{Name: testStepKindName})
				return r
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := testCase.setup(t)
			reg, err := r.Get(testStepKindName)
			require.NoError(t, err)
			require.Equal(t, uint32(1), reg.Metadata.DefaultErrorThreshold)
		})
	}
}
