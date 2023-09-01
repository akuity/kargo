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
		newState   api.StageState
		assertions func(newStateIn, newStateOut api.StageState, err error)
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
							api.StageState,
						) (api.StageState, error) {
							return api.StageState{}, errors.New("something went wrong")
						},
					},
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
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
							newState api.StageState,
						) (api.StageState, error) {
							// This is not a realistic change that a child promotion mechanism
							// would make, but for testing purposes, this is good enough to
							// help us assert that the function under test does return all
							// modifications made by its child promotion mechanisms.
							newState.ID = "fake-mutated-id"
							return newState, nil
						},
					},
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.NoError(t, err)
				// Verify that changes made by child promotion mechanism are returned
				require.Equal(t, "fake-mutated-id", newStateOut.ID)
				// Everything else should be unchanged
				newStateOut.ID = newStateIn.ID
				require.Equal(t, newStateIn, newStateOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStateOut, err := testCase.promoMech.Promote(
				context.Background(),
				&api.Stage{},
				testCase.newState,
			)
			testCase.assertions(testCase.newState, newStateOut, err)
		})
	}
}
