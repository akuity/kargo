package promotionrollouts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCreateRequest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		req    createPromotionRolloutRequest
		fields []string
	}{
		{
			name: "valid",
			req:  createPromotionRolloutRequest{Freight: "abc123"},
		},
		{
			name:   "freight is required",
			fields: []string{"freight"},
		},
		{
			name:   "freight must be a DNS subdomain",
			req:    createPromotionRolloutRequest{Freight: "a_b"},
			fields: []string{"freight"},
		},
		{
			name: "valid with Targets",
			req:  createPromotionRolloutRequest{Freight: "abc123", Targets: []string{"us-east", "us-west"}},
		},
		{
			name:   "Targets must be DNS subdomains, named once",
			req:    createPromotionRolloutRequest{Freight: "abc123", Targets: []string{"us-east", "US", "us-east"}},
			fields: []string{"targets[1]", "targets[2]"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var fields []string
			for _, err := range validateCreateRequest(testCase.req) {
				fields = append(fields, err.Field)
			}
			require.Equal(t, testCase.fields, fields)
		})
	}
}
