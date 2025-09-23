package clusterpromotiontask

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_webhook_ValidateSpec(t *testing.T) {
	testCases := []struct {
		name       string
		spec       kargoapi.PromotionTaskSpec
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "invalid",
			spec: kargoapi.PromotionTaskSpec{
				Steps: []kargoapi.PromotionStep{
					{As: "step-42"}, // This step alias matches a reserved pattern
					{As: "commit"},
					{As: "commit"}, // Duplicate!
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				// We really want to see that all underlying errors have been bubbled up
				// to this level and been aggregated.
				require.Equal(
					t,
					field.ErrorList{
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.steps[0].as",
							BadValue: "step-42",
							Detail:   "step alias is reserved",
						},
						{
							Type:     field.ErrorTypeInvalid,
							Field:    "spec.steps[2].as",
							BadValue: "commit",
							Detail:   "step alias duplicates that of spec.steps[1]",
						},
					},
					errs,
				)
			},
		},
		{
			name: "valid",
			spec: kargoapi.PromotionTaskSpec{
				Steps: []kargoapi.PromotionStep{
					{As: "foo"},
					{As: "bar"},
					{As: "baz"},
					{As: ""},
					{As: ""}, // optional not dup
				},
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				w.validateSpec(field.NewPath("spec"), testCase.spec),
			)
		})
	}
}
