package promote

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromotionOptionsValidate(t *testing.T) {
	testCases := []struct {
		name       string
		opts       promotionOptions
		assertions func(*testing.T, promotionOptions, error)
	}{
		{
			name: "missing project",
			opts: promotionOptions{
				FreightName: "fake-freight",
				Stage:       "fake-stage",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.ErrorContains(t, err, "project is required")
			},
		},
		{
			name: "origin requires stage",
			opts: promotionOptions{
				Project:        "fake-project",
				Origin:         "Warehouse/fake-warehouse",
				DownstreamFrom: "fake-stage",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.ErrorContains(t, err, "origin can only be used with stage")
			},
		},
		{
			name: "origin with stage",
			opts: promotionOptions{
				Project: "fake-project",
				Origin:  "Warehouse/fake-warehouse",
				Stage:   "fake-stage",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.opts.validate()
			testCase.assertions(t, testCase.opts, err)
		})
	}
}
