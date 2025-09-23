package garbage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestCleanProjectPromotions(t *testing.T) {
	testCases := []struct {
		name       string
		collector  *collector
		assertions func(*testing.T, error)
	}{
		{
			name: "error listing Stages",
			collector: &collector{
				listStagesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error listing Stages in Project")
				require.ErrorContains(t, err, "something went wrong")
			},
		},

		{
			name: "error cleaning Stage Promotions",
			collector: &collector{
				listStagesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					stages, ok := objList.(*kargoapi.StageList)
					require.True(t, ok)
					stages.Items = []kargoapi.Stage{{}}
					return nil
				},
				cleanStagePromotionsFn: func(context.Context, string, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error cleaning Promotions to one or more Stages")
			},
		},

		{
			name: "success",
			collector: &collector{
				listStagesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					stages, ok := objList.(*kargoapi.StageList)
					require.True(t, ok)
					stages.Items = []kargoapi.Stage{}
					return nil
				},
				cleanStagePromotionsFn: func(context.Context, string, string) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.collector.cfg.MaxRetainedPromotions = 20
			testCase.assertions(
				t,
				testCase.collector.cleanProjectPromotions(
					context.Background(),
					"fake-project",
				),
			)
		})
	}
}

func TestCleanStagePromotions(t *testing.T) {
	testCases := []struct {
		name       string
		collector  *collector
		assertions func(*testing.T, error)
	}{
		{
			name: "error listing Promotions",
			collector: &collector{
				listPromotionsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error listing Promotions to Stage")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "fewer Promotions threshold",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotions: 2,
				},
				listPromotionsFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := objList.(*kargoapi.PromotionList)
					require.True(t, ok)
					promos.Items = []kargoapi.Promotion{{}}
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error deleting Promotion",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotions:   1,
					MinPromotionDeletionAge: time.Minute,
				},
				listPromotionsFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := objList.(*kargoapi.PromotionList)
					require.True(t, ok)
					now := metav1.Now()
					promos.Items = []kargoapi.Promotion{
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseSucceeded,
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseSucceeded,
							},
						},
					}
					return nil
				},
				deletePromotionFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error deleting one or more Promotions from Stage")
			},
		},
		{
			name: "success",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotions:   1,
					MinPromotionDeletionAge: time.Minute,
				},
				listPromotionsFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := objList.(*kargoapi.PromotionList)
					require.True(t, ok)
					now := metav1.Now()
					promos.Items = []kargoapi.Promotion{
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseSucceeded,
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseSucceeded,
							},
						},
					}
					return nil
				},
				deletePromotionFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.collector.cleanStagePromotions(
					context.Background(),
					"fake-project",
					"fake-stage",
				),
			)
		})
	}
}
