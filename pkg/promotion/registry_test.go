package promotion

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStepRunnerRegistry_Register(t *testing.T) {
	const testStepKind = "fake-step"
	testRegistration := StepRunnerRegistration{
		Metadata: StepRunnerMetadata{
			DefaultTimeout:        time.Duration(0),
			DefaultErrorThreshold: uint32(1),
		},
		Factory: func(StepRunnerCapabilities) StepRunner { return nil },
	}
	testCases := []struct {
		name                 string
		stepKind             string
		registration         StepRunnerRegistration
		expectedRegistration StepRunnerRegistration
		expectedPanic        string
	}{
		{
			name:          "empty step kind panics",
			stepKind:      "",
			expectedPanic: "step kind must be specified",
		},
		{
			name:          "nil Factory function panics",
			stepKind:      testStepKind,
			registration:  StepRunnerRegistration{Factory: nil},
			expectedPanic: "step registration must specify a factory function",
		},
		{
			name:                 "defaults not needed",
			stepKind:             testStepKind,
			registration:         testRegistration,
			expectedRegistration: testRegistration,
		},
		{
			name:     "defaults are applied",
			stepKind: testStepKind,
			registration: StepRunnerRegistration{
				Metadata: StepRunnerMetadata{},
				Factory:  func(StepRunnerCapabilities) StepRunner { return nil },
			},
			expectedRegistration: testRegistration,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			registry := stepRunnerRegistry{}
			if testCase.expectedPanic != "" {
				require.PanicsWithValue(
					t,
					testCase.expectedPanic,
					func() { registry.register(testCase.stepKind, testCase.registration) },
				)
				return
			}
			registry.register(testCase.stepKind, testCase.registration)
			registration := registry[testCase.stepKind]
			require.Equal(t, testCase.expectedRegistration.Metadata, registration.Metadata)
			require.NotNil(t, registration.Factory)
		})
	}
}

func TestStepRunnerRegistry_GetStepRunnerRegistration(t *testing.T) {
	const testStepKind = "fake-step"
	testRegistration := StepRunnerRegistration{
		Metadata: StepRunnerMetadata{
			DefaultTimeout:        5 * time.Minute,
			DefaultErrorThreshold: uint32(3),
			RequiredCapabilities: []StepRunnerCapability{
				StepCapabilityAccessCredentials,
			},
		},
		Factory: func(StepRunnerCapabilities) StepRunner { return nil },
	}
	testCases := []struct {
		name                 string
		setupRegistry        func() stepRunnerRegistry
		expectedRegistration *StepRunnerRegistration
	}{
		{
			name:                 "registration not found",
			setupRegistry:        func() stepRunnerRegistry { return stepRunnerRegistry{} },
			expectedRegistration: nil,
		},
		{
			name: "registration found",
			setupRegistry: func() stepRunnerRegistry {
				registry := stepRunnerRegistry{}
				registry[testStepKind] = testRegistration
				return registry
			},
			expectedRegistration: &testRegistration,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			registry := testCase.setupRegistry()
			registration := registry.getStepRunnerRegistration(testStepKind)
			if testCase.expectedRegistration == nil {
				require.Nil(t, registration)
				return
			}
			require.Equal(
				t,
				testCase.expectedRegistration.Metadata,
				registration.Metadata,
			)
			require.NotNil(t, registration.Factory)
		})
	}
}
