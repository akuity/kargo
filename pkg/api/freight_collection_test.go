package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestBuildTargetFreightCollection(t *testing.T) {
	origin := func(name string) kargoapi.FreightOrigin {
		return kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: name,
		}
	}
	ref := func(freight, warehouse string) kargoapi.FreightReference {
		return kargoapi.FreightReference{Name: freight, Origin: origin(warehouse)}
	}
	requested := func(warehouses ...string) []kargoapi.FreightRequest {
		reqs := make([]kargoapi.FreightRequest, 0, len(warehouses))
		for _, w := range warehouses {
			reqs = append(reqs, kargoapi.FreightRequest{Origin: origin(w)})
		}
		return reqs
	}
	collection := func(refs ...kargoapi.FreightReference) *kargoapi.FreightCollection {
		c := &kargoapi.FreightCollection{}
		c.UpdateOrPush(refs...)
		return c
	}
	names := func(c *kargoapi.FreightCollection) map[string]string {
		out := make(map[string]string, len(c.Freight))
		for o, f := range c.Freight {
			out[o] = f.Name
		}
		return out
	}

	promoted := ref("new-images", "images")

	testCases := []struct {
		name     string
		stage    *kargoapi.Stage
		expected map[string]string
	}{
		{
			name: "last Promotion without a collection is not inherited from",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{RequestedFreight: requested("images", "config")},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{Status: nil},
				},
			},
			expected: map[string]string{"Warehouse/images": "new-images"},
		},
		{
			name: "other origins are carried over from the last Promotion",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{RequestedFreight: requested("images", "config")},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							FreightCollection: collection(
								ref("old-images", "images"),
								ref("cfg", "config"),
							),
						},
					},
				},
			},
			expected: map[string]string{
				"Warehouse/images": "new-images",
				"Warehouse/config": "cfg",
			},
		},
		{
			name: "origins no longer requested are dropped",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{RequestedFreight: requested("images", "config")},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							FreightCollection: collection(
								ref("old-images", "images"),
								ref("cfg", "config"),
								ref("stale", "retired"),
							),
						},
					},
				},
			},
			expected: map[string]string{
				"Warehouse/images": "new-images",
				"Warehouse/config": "cfg",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := buildTargetFreightCollection(context.Background(), promoted, testCase.stage)
			require.NotNil(t, result)
			require.Equal(t, testCase.expected, names(result))
			require.NotEmpty(t, result.ID)
		})
	}
}
