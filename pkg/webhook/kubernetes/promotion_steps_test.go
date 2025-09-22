package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestValidatePromotionSteps(t *testing.T) {
	testCases := []struct {
		name       string
		steps      []kargoapi.PromotionStep
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "steps are valid",
			steps: []kargoapi.PromotionStep{
				{},
				{}, // optional not dup
				{As: "fake-step"},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "steps are invalid",
			steps: []kargoapi.PromotionStep{
				{},
				{As: "step-42"}, // This step alias matches a reserved pattern
				{As: "commit"},
				{As: "commit"}, // Duplicate!
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "steps[1].as",
							BadValue: "step-42",
							Detail:   "step alias is reserved",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "steps[3].as",
							BadValue: "commit",
							Detail:   "step alias duplicates that of steps[2]",
						},
					},
					errs,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				ValidatePromotionSteps(field.NewPath("steps"), testCase.steps),
			)
		})
	}
}
