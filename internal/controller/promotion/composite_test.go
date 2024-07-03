package promotion

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
		freight    []kargoapi.FreightReference
		assertions func(
			t *testing.T,
			promoStatus *kargoapi.PromotionStatus,
			updatedFreight []kargoapi.FreightReference,
			err error,
		)
	}{
		{
			name: "error executing child promotion mechanism",
			promoMech: &compositeMechanism{
				childMechanisms: []Mechanism{
					&FakeMechanism{
						Name: "fake promotion mechanism",
						PromoteFn: func(
							context.Context,
							*kargoapi.Stage,
							[]kargoapi.FreightReference,
						) (*kargoapi.PromotionStatus, []kargoapi.FreightReference, error) {
							return &kargoapi.PromotionStatus{},
								[]kargoapi.FreightReference{},
								errors.New("something went wrong")
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				_ []kargoapi.FreightReference,
				err error,
			) {
				require.ErrorContains(t, err, "error executing fake promotion mechanism")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			freight: []kargoapi.FreightReference{{
				Name: "fake-id",
			}},
			promoMech: &compositeMechanism{
				childMechanisms: []Mechanism{
					&FakeMechanism{
						Name: "fake promotion mechanism",
						PromoteFn: func(
							_ context.Context,
							_ *kargoapi.Stage,
							newFreight []kargoapi.FreightReference,
						) (*kargoapi.PromotionStatus, []kargoapi.FreightReference, error) {
							require.True(t, len(newFreight) > 0)
							// This is not a realistic change that a child promotion mechanism
							// would make, but for testing purposes, this is good enough to
							// help us assert that the function under test does return all
							// modifications made by its child promotion mechanisms.
							newFreight[0].Name = "fake-mutated-id"
							return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded}, newFreight, nil
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				updatedFreight []kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				// Verify that changes made by child promotion mechanism are returned
				require.Equal(
					t,
					[]kargoapi.FreightReference{{
						Name: "fake-mutated-id",
					}},
					updatedFreight,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			promoStatus, updatedFreight, err := testCase.promoMech.Promote(
				context.Background(),
				&kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					},
				},
				&kargoapi.Promotion{},
				testCase.freight,
			)
			testCase.assertions(t, promoStatus, updatedFreight, err)
		})
	}
}
