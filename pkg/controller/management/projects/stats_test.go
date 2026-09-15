package projects

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
)

func Test_reconciler_collectStats(t *testing.T) {
	const testProject = "fake-project"

	scheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		project    *kargoapi.Project
		client     client.Client
		assertions func(*testing.T, kargoapi.ProjectStatus, error)
	}{
		{
			name:    "Project not ready",
			project: &kargoapi.Project{},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				require.Nil(t, status.Stats)
			},
		},
		{
			name: "error listing Warehouses",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(
						context.Context,
						client.WithWatch,
						client.ObjectList,
						...client.ListOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				}).Build(),
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.Error(t, err)
				cond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, cond)
				require.Equal(t, metav1.ConditionFalse, cond.Status)
				require.Equal(t, kargoapi.ConditionTypeHealthy, cond.Type)
				require.Equal(t, "CollectingWarehouseStatsFailed", cond.Reason)
			},
		},
		{
			name: "error listing Stages",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(
						_ context.Context,
						_ client.WithWatch,
						list client.ObjectList,
						_ ...client.ListOption,
					) error {
						if _, ok := list.(*kargoapi.StageList); ok {
							return fmt.Errorf("something went wrong")
						}
						return nil
					},
				}).Build(),
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.Error(t, err)
				cond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, cond)
				require.Equal(t, metav1.ConditionFalse, cond.Status)
				require.Equal(t, kargoapi.ConditionTypeHealthy, cond.Type)
				require.Equal(t, "CollectingStageStatsFailed", cond.Reason)
			},
		},
		{
			name: "error listing Targets",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(
						_ context.Context,
						_ client.WithWatch,
						list client.ObjectList,
						_ ...client.ListOption,
					) error {
						if _, ok := list.(*kargoapi.TargetList); ok {
							return fmt.Errorf("something went wrong")
						}
						return nil
					},
				}).Build(),
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.Error(t, err)
				cond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, cond)
				require.Equal(t, metav1.ConditionFalse, cond.Status)
				require.Equal(t, kargoapi.ConditionTypeHealthy, cond.Type)
				require.Equal(t, "CollectingTargetStatsFailed", cond.Reason)
			},
		},
		{
			name: "error listing PromotionRequests",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(
						_ context.Context,
						_ client.WithWatch,
						list client.ObjectList,
						_ ...client.ListOption,
					) error {
						if _, ok := list.(*kargoapi.PromotionRequestList); ok {
							return fmt.Errorf("something went wrong")
						}
						return nil
					},
				}).Build(),
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.Error(t, err)
				cond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, cond)
				require.Equal(t, metav1.ConditionFalse, cond.Status)
				require.Equal(t, kargoapi.ConditionTypeHealthy, cond.Type)
				require.Equal(t, "CollectingTargetStatsFailed", cond.Reason)
			},
		},
		{
			name: "successful stats collection",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "warehouse1",
						Namespace: testProject,
					},
					Status: kargoapi.WarehouseStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionTrue,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "warehouse2",
						Namespace: testProject,
					},
					Status: kargoapi.WarehouseStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionFalse,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "warehouse3",
						Namespace: testProject,
					},
					// No health condition == unknown
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "warehouse4",
						Namespace: testProject,
					},
					Status: kargoapi.WarehouseStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionStatus("bogus"),
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "warehouse5",
						Namespace: testProject,
					},
					Status: kargoapi.WarehouseStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionUnknown,
						}},
					},
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stage1",
						Namespace: testProject,
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
					Status: kargoapi.StageStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionTrue,
						}},
					},
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stage2",
						Namespace: testProject,
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
					Status: kargoapi.StageStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionFalse,
						}},
					},
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stage3-control-flow",
						Namespace: testProject,
					},
					// No health condition == unknown
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stage4",
						Namespace: testProject,
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
					Status: kargoapi.StageStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionStatus("bogus"), // Unknown
						}},
					},
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stage5",
						Namespace: testProject,
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
					Status: kargoapi.StageStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionUnknown,
						}},
					},
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fleet",
						Namespace: testProject,
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
						Targets: &kargoapi.StageTargets{
							Selectors: []metav1.LabelSelector{{}},
						},
					},
					Status: kargoapi.StageStatus{
						LastPromotionRequest: &kargoapi.PromotionRequestReference{
							Name: "fleet.01",
						},
					},
				},
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fleet.01",
						Namespace: testProject,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "fleet",
						Freight: "f",
						Targets: []kargoapi.PromotionRequestTarget{{Name: "t1"}, {Name: "t2"}},
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase:   kargoapi.PromotionRequestPhaseErrored,
						Summary: &kargoapi.PromotionRequestSummary{Succeeded: 1, Errored: 1},
					},
				},
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{Name: "t1", Namespace: testProject},
				},
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{Name: "t2", Namespace: testProject},
				},
			).Build(),
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				require.Nil(t, conditions.Get(&status, kargoapi.ConditionTypeHealthy))
				stats := status.Stats
				require.Equal(t, int64(5), stats.Warehouses.Count)
				require.Equal(t, int64(1), stats.Warehouses.Health.Healthy)
				require.Equal(t, int64(5), stats.Stages.Count)
				require.Equal(t, int64(1), stats.Stages.Health.Healthy)
				require.Equal(
					t,
					&kargoapi.TargetStats{
						Count:     2,
						Promotion: kargoapi.PromotionRequestSummary{Succeeded: 1, Errored: 1},
					},
					stats.Targets,
				)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{client: tt.client}
			status, err := r.collectStats(t.Context(), tt.project)
			tt.assertions(t, status, err)
		})
	}
}

func Test_collectTargetStats(t *testing.T) {
	const testProject = "fake-project"
	targetAware := func(name string, current, last string) kargoapi.Stage {
		stage := kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testProject},
			Spec: kargoapi.StageSpec{
				Targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{{}}},
			},
		}
		if current != "" {
			stage.Status.CurrentPromotionRequest = &kargoapi.PromotionRequestReference{Name: current}
		}
		if last != "" {
			stage.Status.LastPromotionRequest = &kargoapi.PromotionRequestReference{Name: last}
		}
		return stage
	}
	classic := func(name string) kargoapi.Stage {
		return kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testProject},
			Status: kargoapi.StageStatus{
				// A classic Stage never has these, but even if it did, it must
				// not be counted.
				LastPromotionRequest: &kargoapi.PromotionRequestReference{Name: "ignored"},
			},
		}
	}
	request := func(
		name string,
		targets int,
		phase kargoapi.PromotionRequestPhase,
		summary *kargoapi.PromotionRequestSummary,
	) kargoapi.PromotionRequest {
		req := kargoapi.PromotionRequest{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testProject},
			Status:     kargoapi.PromotionRequestStatus{Phase: phase, Summary: summary},
		}
		for i := 0; i < targets; i++ {
			req.Spec.Targets = append(req.Spec.Targets, kargoapi.PromotionRequestTarget{Name: "t"})
		}
		return req
	}
	targets := func(n int) []kargoapi.Target {
		out := make([]kargoapi.Target, n)
		for i := range out {
			out[i].Name = "t"
		}
		return out
	}

	testCases := []struct {
		name     string
		targets  []kargoapi.Target
		stages   []kargoapi.Stage
		requests []kargoapi.PromotionRequest
		expected *kargoapi.TargetStats
	}{
		{
			name:     "no Targets and no target-aware Stages",
			stages:   []kargoapi.Stage{classic("classic")},
			requests: []kargoapi.PromotionRequest{request("ignored", 3, kargoapi.PromotionRequestPhaseSucceeded, nil)},
			expected: nil,
		},
		{
			name:     "Targets but no target-aware Stages",
			targets:  targets(3),
			stages:   []kargoapi.Stage{classic("classic")},
			expected: &kargoapi.TargetStats{Count: 3},
		},
		{
			name:     "target-aware Stage that has never promoted",
			targets:  targets(2),
			stages:   []kargoapi.Stage{targetAware("fleet", "", "")},
			expected: &kargoapi.TargetStats{Count: 2},
		},
		{
			name:    "sums the recorded summary of each Stage's latest request",
			targets: targets(5),
			stages: []kargoapi.Stage{
				targetAware("a", "", "a.01"),
				targetAware("b", "", "b.01"),
			},
			requests: []kargoapi.PromotionRequest{
				request("a.01", 3, kargoapi.PromotionRequestPhaseSucceeded,
					&kargoapi.PromotionRequestSummary{Succeeded: 3}),
				request("b.01", 2, kargoapi.PromotionRequestPhaseErrored,
					&kargoapi.PromotionRequestSummary{Succeeded: 1, Errored: 1}),
			},
			expected: &kargoapi.TargetStats{
				Count:     5,
				Promotion: kargoapi.PromotionRequestSummary{Succeeded: 4, Errored: 1},
			},
		},
		{
			name:    "prefers the current request over the last",
			targets: targets(2),
			stages:  []kargoapi.Stage{targetAware("a", "a.02", "a.01")},
			requests: []kargoapi.PromotionRequest{
				request("a.01", 2, kargoapi.PromotionRequestPhaseSucceeded,
					&kargoapi.PromotionRequestSummary{Succeeded: 2}),
				request("a.02", 2, kargoapi.PromotionRequestPhaseRunning,
					&kargoapi.PromotionRequestSummary{Running: 1, Succeeded: 1}),
			},
			expected: &kargoapi.TargetStats{
				Count:     2,
				Promotion: kargoapi.PromotionRequestSummary{Running: 1, Succeeded: 1},
			},
		},
		{
			name:    "a request without a summary counts its Targets by its own phase",
			targets: targets(4),
			stages: []kargoapi.Stage{
				targetAware("pending", "p.01", ""),
				targetAware("errored", "", "e.01"),
				targetAware("failed", "", "f.01"),
				targetAware("succeeded", "", "s.01"),
			},
			requests: []kargoapi.PromotionRequest{
				request("p.01", 2, kargoapi.PromotionRequestPhasePending, nil),
				request("e.01", 3, kargoapi.PromotionRequestPhaseErrored, nil),
				request("f.01", 1, kargoapi.PromotionRequestPhaseFailed, nil),
				request("s.01", 4, kargoapi.PromotionRequestPhaseSucceeded, nil),
			},
			expected: &kargoapi.TargetStats{
				Count: 4,
				Promotion: kargoapi.PromotionRequestSummary{
					Pending: 2, Errored: 3, Failed: 1, Succeeded: 4,
				},
			},
		},
		{
			name:    "a latest request that no longer exists is a missing round",
			targets: targets(1),
			stages: []kargoapi.Stage{
				targetAware("gone", "", "gone.01"),
				targetAware("here", "", "here.01"),
			},
			requests: []kargoapi.PromotionRequest{
				request("here.01", 1, kargoapi.PromotionRequestPhaseSucceeded,
					&kargoapi.PromotionRequestSummary{Succeeded: 1}),
			},
			expected: &kargoapi.TargetStats{
				Count:     1,
				Promotion: kargoapi.PromotionRequestSummary{Succeeded: 1},
				Unknown:   1,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				testCase.expected,
				collectTargetStats(testCase.targets, testCase.stages, testCase.requests),
			)
		})
	}
}
