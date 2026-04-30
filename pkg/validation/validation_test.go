package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestSemverConstraint(t *testing.T) {
	testCases := []struct {
		name             string
		semverConstraint string
		assertions       func(*testing.T, error)
	}{
		{
			name: "empty string",
			assertions: func(t *testing.T, err error) {
				require.Nil(t, err)
			},
		},

		{
			name:             "invalid",
			semverConstraint: "bogus",
			assertions: func(t *testing.T, err error) {
				require.NotNil(t, err)
				require.Equal(
					t,
					&field.Error{
						Type:     field.ErrorTypeInvalid,
						Field:    "semverConstraint",
						BadValue: "bogus",
					},
					err,
				)
			},
		},

		{
			name:             "valid",
			semverConstraint: "^1.0.0",
			assertions: func(t *testing.T, err error) {
				require.Nil(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				SemverConstraint(
					field.NewPath("semverConstraint"),
					testCase.semverConstraint,
				),
			)
		})
	}
}
