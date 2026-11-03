package promotionrollouts

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
		{
			name: "valid with an update strategy",
			req: createPromotionRolloutRequest{
				Freight: "abc123",
				UpdateStrategy: &kargoapi.TargetUpdateStrategy{
					MaxConcurrent: new(intstr.FromInt32(1)),
					MaxFailures:   new(intstr.FromString("0%")),
				},
			},
		},
		{
			name: "an update strategy may give either value alone",
			req: createPromotionRolloutRequest{
				Freight:        "abc123",
				UpdateStrategy: &kargoapi.TargetUpdateStrategy{MaxFailures: new(intstr.FromInt32(0))},
			},
		},
		{
			name: "update strategy values have the Stage's floors",
			req: createPromotionRolloutRequest{
				Freight: "abc123",
				UpdateStrategy: &kargoapi.TargetUpdateStrategy{
					MaxConcurrent: new(intstr.FromString("0%")),
					MaxFailures:   new(intstr.FromInt32(-1)),
				},
			},
			fields: []string{"updateStrategy.maxConcurrent", "updateStrategy.maxFailures"},
		},
		{
			name: "update strategy values must be numbers or percentages",
			req: createPromotionRolloutRequest{
				Freight:        "abc123",
				UpdateStrategy: &kargoapi.TargetUpdateStrategy{MaxConcurrent: new(intstr.FromString("half"))},
			},
			fields: []string{"updateStrategy.maxConcurrent"},
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
