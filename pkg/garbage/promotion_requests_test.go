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

func TestCleanProjectPromotionRequests(t *testing.T) {
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
			name: "error cleaning Stage PromotionRequests",
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
				cleanStagePromotionRequestsFn: func(context.Context, string, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, "error cleaning PromotionRequests for one or more Stages",
				)
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
				cleanStagePromotionRequestsFn: func(context.Context, string, string) error {
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
			testCase.collector.cfg.MaxRetainedPromotionRequests = 20
			testCase.assertions(
				t,
				testCase.collector.cleanProjectPromotionRequests(
					t.Context(),
					"fake-project",
				),
			)
		})
	}
}

func TestCleanStagePromotionRequests(t *testing.T) {
	// listPromotionRequestsFn returns PromotionRequests created at hourly
	// intervals, oldest last, each in the corresponding phase.
	listRequestsWithPhases := func(
		phases ...kargoapi.PromotionRequestPhase,
	) func(context.Context, client.ObjectList, ...client.ListOption) error {
		return func(
			_ context.Context,
			objList client.ObjectList,
			_ ...client.ListOption,
		) error {
			requests, ok := objList.(*kargoapi.PromotionRequestList)
			require.True(t, ok)
			now := metav1.Now()
			requests.Items = make([]kargoapi.PromotionRequest, len(phases))
			for i, phase := range phases {
				requests.Items[i] = kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Name: string(rune('a' + i)),
						CreationTimestamp: metav1.NewTime(
							now.Add(-time.Duration(i+1) * time.Hour),
						),
					},
					Status: kargoapi.PromotionRequestStatus{Phase: phase},
				}
			}
			return nil
		}
	}

	testCases := []struct {
		name       string
		collector  *collector
		assertions func(*testing.T, []string, error)
	}{
		{
			name: "error listing PromotionRequests",
			collector: &collector{
				listPromotionRequestsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(
					t, err, "error listing PromotionRequests for Stage",
				)
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "fewer PromotionRequests than threshold",
			collector: &collector{
				cfg: CollectorConfig{MaxRetainedPromotionRequests: 2},
				listPromotionRequestsFn: listRequestsWithPhases(
					kargoapi.PromotionRequestPhaseSucceeded,
				),
			},
			assertions: func(t *testing.T, deleted []string, err error) {
				require.NoError(t, err)
				require.Empty(t, deleted)
			},
		},
		{
			name: "non-terminal PromotionRequests are never deleted",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotionRequests:   1,
					MinPromotionRequestDeletionAge: time.Minute,
				},
				// The oldest is still Running, so nothing older than it exists to
				// delete and the whole list is spared.
				listPromotionRequestsFn: listRequestsWithPhases(
					kargoapi.PromotionRequestPhaseSucceeded,
					kargoapi.PromotionRequestPhaseSucceeded,
					kargoapi.PromotionRequestPhaseRunning,
				),
			},
			assertions: func(t *testing.T, deleted []string, err error) {
				require.NoError(t, err)
				require.Empty(t, deleted)
			},
		},
		{
			name: "retains the newest N older than the oldest non-terminal",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotionRequests:   1,
					MinPromotionRequestDeletionAge: time.Minute,
				},
				// "b" is Pending, so "c" is spared as the one retained request
				// older than it, and only "d" is eligible.
				listPromotionRequestsFn: listRequestsWithPhases(
					kargoapi.PromotionRequestPhaseSucceeded,
					kargoapi.PromotionRequestPhasePending,
					kargoapi.PromotionRequestPhaseFailed,
					kargoapi.PromotionRequestPhaseErrored,
				),
			},
			assertions: func(t *testing.T, deleted []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"d"}, deleted)
			},
		},
		{
			name: "PromotionRequests younger than the minimum age are spared",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotionRequests:   1,
					MinPromotionRequestDeletionAge: 24 * time.Hour,
				},
				listPromotionRequestsFn: listRequestsWithPhases(
					kargoapi.PromotionRequestPhaseSucceeded,
					kargoapi.PromotionRequestPhaseSucceeded,
				),
			},
			assertions: func(t *testing.T, deleted []string, err error) {
				require.NoError(t, err)
				require.Empty(t, deleted)
			},
		},
		{
			name: "error deleting PromotionRequest",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotionRequests:   1,
					MinPromotionRequestDeletionAge: time.Minute,
				},
				listPromotionRequestsFn: listRequestsWithPhases(
					kargoapi.PromotionRequestPhaseSucceeded,
					kargoapi.PromotionRequestPhaseSucceeded,
				),
				deletePromotionRequestFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(
					t,
					err,
					"error deleting one or more PromotionRequests for Stage",
				)
			},
		},
		{
			name: "success",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotionRequests:   1,
					MinPromotionRequestDeletionAge: time.Minute,
				},
				listPromotionRequestsFn: listRequestsWithPhases(
					kargoapi.PromotionRequestPhaseSucceeded,
					kargoapi.PromotionRequestPhaseSucceeded,
				),
			},
			assertions: func(t *testing.T, deleted []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"b"}, deleted)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var deleted []string
			if testCase.collector.deletePromotionRequestFn == nil {
				testCase.collector.deletePromotionRequestFn = func(
					_ context.Context,
					obj client.Object,
					_ ...client.DeleteOption,
				) error {
					deleted = append(deleted, obj.GetName())
					return nil
				}
			}
			err := testCase.collector.cleanStagePromotionRequests(
				t.Context(),
				"fake-project",
				"fake-stage",
			)
			testCase.assertions(t, deleted, err)
		})
	}
}
