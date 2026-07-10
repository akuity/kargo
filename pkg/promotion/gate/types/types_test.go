package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestPromotionInput_FreightRequest(t *testing.T) {
	t.Parallel()

	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "test-warehouse",
	}
	otherOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "other-warehouse",
	}
	matchingRequest := kargoapi.FreightRequest{
		Origin:  testOrigin,
		Sources: kargoapi.FreightSources{Stages: []string{"upstream"}},
	}

	stageWith := func(requests ...kargoapi.FreightRequest) *kargoapi.Stage {
		return &kargoapi.Stage{
			Spec: kargoapi.StageSpec{RequestedFreight: requests},
		}
	}
	freightWith := func(origin kargoapi.FreightOrigin) *kargoapi.Freight {
		return &kargoapi.Freight{Origin: origin}
	}

	testCases := []struct {
		name     string
		input    PromotionInput
		expected *kargoapi.FreightRequest
	}{
		{
			name:  "nil Stage",
			input: PromotionInput{Freight: freightWith(testOrigin)},
		},
		{
			name:  "nil Freight",
			input: PromotionInput{Stage: stageWith(matchingRequest)},
		},
		{
			name: "no matching origin",
			input: PromotionInput{
				Stage:   stageWith(kargoapi.FreightRequest{Origin: otherOrigin}),
				Freight: freightWith(testOrigin),
			},
		},
		{
			name: "matching origin",
			input: PromotionInput{
				Stage:   stageWith(matchingRequest),
				Freight: freightWith(testOrigin),
			},
			expected: &matchingRequest,
		},
		{
			name: "returns first matching origin",
			input: PromotionInput{
				Stage: stageWith(
					kargoapi.FreightRequest{Origin: otherOrigin},
					matchingRequest,
				),
				Freight: freightWith(testOrigin),
			},
			expected: &matchingRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.input.FreightRequest()
			if testCase.expected == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, *testCase.expected, *got)
		})
	}
}
