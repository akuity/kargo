package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/promotion"
)

// validationTestCase is a struct that represents a test case for validating
// a promotion step configuration. It includes the name of the test,
// the configuration to validate, and the expected problems that should
// be reported if the validation fails.
type validationTestCase struct {
	name             string
	config           promotion.Config
	expectedProblems []string
}

// runValidationTests runs a set of validation tests for a given promotion step
// configuration converter function. It takes a testing.T instance, a converter
// function that converts a promotion.Config to a specific type T, and a slice
// of validationTestCase instances. Each test case is run in parallel, and the
// results are asserted using require and assert from the testify package.
func runValidationTests[T any](t *testing.T, converter func(promotion.Config) (T, error), tests []validationTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := converter(tt.config)
			if len(tt.expectedProblems) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, problem := range tt.expectedProblems {
				assert.ErrorContains(t, err, problem)
			}
		})
	}
}
