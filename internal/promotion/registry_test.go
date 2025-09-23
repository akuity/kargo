package promotion

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/promotion"
)

func TestStepRunnerRegistry_Register(t *testing.T) {
	const testStepKind = "fake-step"
	testRegistration := promotion.StepRunnerRegistration{
		Metadata: promotion.StepRunnerMetadata{
			DefaultTimeout:        time.Duration(0),
			DefaultErrorThreshold: uint32(1),
		},
		Factory: func(promotion.StepRunnerCapabilities) promotion.StepRunner { return nil },
	}
	testCases := []struct {
		name                 string
		stepKind             string
		registration         promotion.StepRunnerRegistration
		expectedRegistration promotion.StepRunnerRegistration
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
			registration:  promotion.StepRunnerRegistration{Factory: nil},
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
			registration: promotion.StepRunnerRegistration{
				Metadata: promotion.StepRunnerMetadata{},
				Factory:  func(promotion.StepRunnerCapabilities) promotion.StepRunner { return nil },
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
	testRegistration := promotion.StepRunnerRegistration{
		Metadata: promotion.StepRunnerMetadata{
			DefaultTimeout:        5 * time.Minute,
			DefaultErrorThreshold: uint32(3),
			RequiredCapabilities: []promotion.StepRunnerCapability{
				promotion.StepCapabilityAccessCredentials,
			},
		},
		Factory: func(promotion.StepRunnerCapabilities) promotion.StepRunner { return nil },
	}
	testCases := []struct {
		name                 string
		setupRegistry        func() stepRunnerRegistry
		expectedRegistration *promotion.StepRunnerRegistration
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
