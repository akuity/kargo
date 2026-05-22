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
			name: "accepts reason for direct stage promotion",
			opts: promotionOptions{
				Project:     "fake-project",
				FreightName: "fake-freight",
				Stage:       "fake-stage",
				Reason:      " rollback ",
			},
			assertions: func(t *testing.T, opts promotionOptions, err error) {
				require.NoError(t, err)
				require.Equal(t, "rollback", opts.Reason)
			},
		},
		{
			name: "rejects reason for downstream promotion",
			opts: promotionOptions{
				Project:        "fake-project",
				FreightName:    "fake-freight",
				DownstreamFrom: "fake-stage",
				Reason:         "rollback",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.ErrorContains(t, err, "reason can only be used")
			},
		},
		{
			name: "accepts expected auto candidate for direct stage promotion",
			opts: promotionOptions{
				Project:               "fake-project",
				FreightName:           "fake-freight",
				Stage:                 "fake-stage",
				ExpectedAutoCandidate: "newer-freight",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "rejects expected auto candidate for downstream promotion",
			opts: promotionOptions{
				Project:               "fake-project",
				FreightName:           "fake-freight",
				DownstreamFrom:        "fake-stage",
				ExpectedAutoCandidate: "newer-freight",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.ErrorContains(t, err, "expected-auto-candidate can only be used")
			},
		},
		{
			name: "rejects reason for abort",
			opts: promotionOptions{
				Project:   "fake-project",
				Promotion: "fake-promotion",
				Abort:     true,
				Reason:    "rollback",
			},
			assertions: func(t *testing.T, _ promotionOptions, err error) {
				require.ErrorContains(t, err, "reason can only be used")
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
