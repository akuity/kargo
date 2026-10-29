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
