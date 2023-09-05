package promotion

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewCompositeMechanism(t *testing.T) {
	testName := "fake-name"
	testChildPromotionMechanisms := []Mechanism{
		&FakeMechanism{},
		&FakeMechanism{},
	}
	pm := newCompositeMechanism(
		testName,
		testChildPromotionMechanisms...,
	)
	cpm, ok := pm.(*compositeMechanism)
	require.True(t, ok)
	require.Equal(t, testName, cpm.name)
	require.Equal(t, testChildPromotionMechanisms, cpm.childMechanisms)
}

func TestCompositeName(t *testing.T) {
	const testName = "fake name"
	require.Equal(
		t,
		testName,
		(&compositeMechanism{
			name: testName,
		}).GetName(),
	)
}

func TestCompositePromote(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *compositeMechanism
		newFreight api.Freight
		assertions func(newFreightIn, newFreightOut api.Freight, err error)
	}{
		{
			name: "error executing child promotion mechanism",
			promoMech: &compositeMechanism{
				childMechanisms: []Mechanism{
					&FakeMechanism{
						Name: "fake promotion mechanism",
						PromoteFn: func(
							context.Context,
							*api.Stage,
							api.Freight,
						) (api.Freight, error) {
							return api.Freight{}, errors.New("something went wrong")
						},
					},
				},
			},
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error executing fake promotion mechanism",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success",
			promoMech: &compositeMechanism{
				childMechanisms: []Mechanism{
					&FakeMechanism{
						Name: "fake promotion mechanism",
						PromoteFn: func(
							_ context.Context,
							_ *api.Stage,
							newFreight api.Freight,
						) (api.Freight, error) {
							// This is not a realistic change that a child promotion mechanism
							// would make, but for testing purposes, this is good enough to
							// help us assert that the function under test does return all
							// modifications made by its child promotion mechanisms.
							newFreight.ID = "fake-mutated-id"
							return newFreight, nil
						},
					},
				},
			},
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.NoError(t, err)
				// Verify that changes made by child promotion mechanism are returned
				require.Equal(t, "fake-mutated-id", newFreightOut.ID)
				// Everything else should be unchanged
				newFreightOut.ID = newFreightIn.ID
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newFreightOut, err := testCase.promoMech.Promote(
				context.Background(),
				&api.Stage{
					Spec: &api.StageSpec{
						PromotionMechanisms: &api.PromotionMechanisms{},
					},
				},
				testCase.newFreight,
			)
			testCase.assertions(testCase.newFreight, newFreightOut, err)
		})
	}
}
