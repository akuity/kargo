package promotion

import (
	"context"
	"errors"
	"fmt"
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
			origFreight *kargoapi.FreightCollection,
			promo *kargoapi.Promotion,
			err error,
		)
	}{
		{
			name: "error executing child promotion mechanism",
			promoMech: &compositeMechanism{
				childMechanisms: []Mechanism{
					&FakeMechanism{
						Name: "fake promotion mechanism",
						PromoteFn: func(context.Context, *kargoapi.Stage, *kargoapi.Promotion) error {
							return errors.New("something went wrong")
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.FreightCollection,
				_ *kargoapi.Promotion,
				err error,
			) {
				require.ErrorContains(t, err, "error executing fake promotion mechanism")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			freight: []kargoapi.FreightReference{{
				Name:    "fake-id",
				Commits: []kargoapi.GitCommit{{}},
			}},
			promoMech: &compositeMechanism{
				childMechanisms: []Mechanism{
					&FakeMechanism{
						Name: "fake promotion mechanism",
						PromoteFn: func(
							_ context.Context,
							_ *kargoapi.Stage,
							promo *kargoapi.Promotion,
						) error {
							refs := promo.Status.FreightCollection.References()
							require.NotEmpty(t, refs)
							require.NotEmpty(t, refs[0].Commits)
							refs[0].Commits[0].HealthCheckCommit = "fake-commit-id"
							promo.Status.FreightCollection.UpdateOrPush(refs[0])
							promo.Status.Phase = kargoapi.PromotionPhaseSucceeded
							return nil
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.FreightCollection,
				promo *kargoapi.Promotion,
				err error,
			) {
				require.NoError(t, err)
				// Verify that changes made by child promotion mechanism are returned
				refs := promo.Status.FreightCollection.References()
				require.NotEmpty(t, refs)
				require.NotEmpty(t, refs[0].Commits)
				require.Equal(t, "fake-commit-id", refs[0].Commits[0].HealthCheckCommit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			promo := &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					FreightCollection: &kargoapi.FreightCollection{},
				},
			}
			for _, freight := range testCase.freight {
				promo.Status.FreightCollection.UpdateOrPush(freight)
			}
			origFreight := promo.Status.FreightCollection.DeepCopy()
			err := testCase.promoMech.Promote(
				context.Background(),
				&kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					},
				},
				promo,
			)
			testCase.assertions(t, origFreight, promo, err)
		})
	}
}

func TestMergePromoStatus(t *testing.T) {
	t.Run("phase merging", func(t *testing.T) {
		testCases := []struct {
			olderPhase    kargoapi.PromotionPhase
			newerPhase    kargoapi.PromotionPhase
			expectedPhase kargoapi.PromotionPhase
		}{
			{
				olderPhase:    kargoapi.PromotionPhaseErrored,
				newerPhase:    kargoapi.PromotionPhaseFailed,
				expectedPhase: kargoapi.PromotionPhaseErrored,
			},
			{
				olderPhase:    kargoapi.PromotionPhaseFailed,
				newerPhase:    kargoapi.PromotionPhaseErrored,
				expectedPhase: kargoapi.PromotionPhaseErrored,
			},
			{
				olderPhase:    kargoapi.PromotionPhaseFailed,
				newerPhase:    kargoapi.PromotionPhaseRunning,
				expectedPhase: kargoapi.PromotionPhaseFailed,
			},
			{
				olderPhase:    kargoapi.PromotionPhaseRunning,
				newerPhase:    kargoapi.PromotionPhaseFailed,
				expectedPhase: kargoapi.PromotionPhaseFailed,
			},
			{
				olderPhase:    kargoapi.PromotionPhaseRunning,
				newerPhase:    kargoapi.PromotionPhaseSucceeded,
				expectedPhase: kargoapi.PromotionPhaseRunning,
			},
			{
				olderPhase:    kargoapi.PromotionPhaseSucceeded,
				newerPhase:    kargoapi.PromotionPhaseRunning,
				expectedPhase: kargoapi.PromotionPhaseRunning,
			},
			{
				olderPhase:    kargoapi.PromotionPhaseSucceeded,
				newerPhase:    kargoapi.PromotionPhaseSucceeded,
				expectedPhase: kargoapi.PromotionPhaseSucceeded,
			},
		}
		for _, testCase := range testCases {
			t.Run(
				fmt.Sprintf("old is %s, new is %s", testCase.olderPhase, testCase.newerPhase),
				func(t *testing.T) {
					mergedStatus := mergePromoStatus(
						&kargoapi.PromotionStatus{Phase: testCase.newerPhase},
						&kargoapi.PromotionStatus{Phase: testCase.olderPhase},
					)
					require.Equal(t, testCase.expectedPhase, mergedStatus.Phase)
				},
			)
		}
	})

	t.Run("freight collection replacement", func(t *testing.T) {
		olderStatus := &kargoapi.PromotionStatus{
			FreightCollection: &kargoapi.FreightCollection{},
		}
		olderStatus.FreightCollection.UpdateOrPush(kargoapi.FreightReference{
			Commits: []kargoapi.GitCommit{{}},
		})
		newerStatus := &kargoapi.PromotionStatus{
			FreightCollection: olderStatus.FreightCollection.DeepCopy(),
		}
		newerStatus.FreightCollection.UpdateOrPush(kargoapi.FreightReference{
			Commits: []kargoapi.GitCommit{{
				HealthCheckCommit: "fake-commit",
			}},
		})
		mergedStatus := mergePromoStatus(newerStatus, olderStatus)
		require.Same(t, newerStatus.FreightCollection, mergedStatus.FreightCollection)
	})

	t.Run("metadata merging", func(t *testing.T) {
		olderStatus := &kargoapi.PromotionStatus{
			Metadata: map[string]string{
				"a": "b",
				"c": "d", // Should be overwritten
			},
		}
		newerStatus := &kargoapi.PromotionStatus{
			Metadata: map[string]string{
				"c": "D", // Should overwrite
				"e": "f",
			},
		}
		mergedStatus := mergePromoStatus(newerStatus, olderStatus)
		require.Equal(t, "b", mergedStatus.Metadata["a"])
		require.Equal(t, "D", mergedStatus.Metadata["c"])
		require.Equal(t, "f", mergedStatus.Metadata["e"])
	})
}
