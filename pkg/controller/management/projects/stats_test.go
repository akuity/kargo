package projects

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/database"
)

// fakeTargetCounter counts a fixed set of Targets the way CountTargets does,
// or fails.
type fakeTargetCounter struct {
	targets []string
	err     error
}

func (f fakeTargetCounter) Count(
	_ context.Context,
	_ string,
	names []string,
) (database.CountTargetsRow, error) {
	if f.err != nil {
		return database.CountTargetsRow{}, f.err
	}
	counts := database.CountTargetsRow{Total: int64(len(f.targets))}
	for _, target := range f.targets {
		if slices.Contains(names, target) {
			counts.Among++
		}
	}
	return counts, nil
}

func Test_reconciler_collectStats(t *testing.T) {
	const testProject = "fake-project"

	scheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		project    *kargoapi.Project
		client     client.Client
		targets    TargetCounter
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
			client:  fake.NewClientBuilder().WithScheme(scheme).Build(),
			targets: fakeTargetCounter{err: fmt.Errorf("something went wrong")},
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
			name: "no database reports no Target stats",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.Stats)
				require.Nil(t, status.Stats.Targets)
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
						Phase: kargoapi.PromotionRequestPhaseErrored,
						Targets: []kargoapi.PromotionRequestTargetStatus{
							{Name: "t1", Promotion: "p1", Phase: kargoapi.PromotionPhaseSucceeded},
							{Name: "t2", Promotion: "p2", Phase: kargoapi.PromotionPhaseErrored},
						},
					},
				},
			).Build(),
			targets: fakeTargetCounter{targets: []string{"t1", "t2"}},
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
						Count:  2,
						Health: kargoapi.HealthStats{Healthy: 1},
					},
					stats.Targets,
				)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{client: tt.client, targets: tt.targets}
			status, err := r.collectStats(t.Context(), tt.project)
			tt.assertions(t, status, err)
		})
	}
}

func Test_targetStats(t *testing.T) {
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
				// A classic Stage never has this, but even if it did, it must
				// not be consulted.
				LastPromotionRequest: &kargoapi.PromotionRequestReference{Name: "ignored"},
			},
		}
	}
	// request builds a PromotionRequest naming the given Targets, with the
	// given per-Target phases recorded in status where provided.
	request := func(
		name string,
		phase kargoapi.PromotionRequestPhase,
		phases map[string]kargoapi.PromotionPhase,
		names ...string,
	) kargoapi.PromotionRequest {
		req := kargoapi.PromotionRequest{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testProject},
			Status:     kargoapi.PromotionRequestStatus{Phase: phase},
		}
		for _, n := range names {
			req.Spec.Targets = append(req.Spec.Targets, kargoapi.PromotionRequestTarget{Name: n})
			if p, ok := phases[n]; ok {
				req.Status.Targets = append(req.Status.Targets, kargoapi.PromotionRequestTargetStatus{
					Name:      n,
					Promotion: "p",
					Phase:     p,
				})
			}
		}
		return req
	}
	const (
		succeeded = kargoapi.PromotionPhaseSucceeded
		errored   = kargoapi.PromotionPhaseErrored
		running   = kargoapi.PromotionPhaseRunning
	)

	testCases := []struct {
		name     string
		targets  []string
		stages   []kargoapi.Stage
		requests []kargoapi.PromotionRequest
		expected *kargoapi.TargetStats
	}{
		{
			name:     "no Targets",
			stages:   []kargoapi.Stage{targetAware("fleet", "", "f.01")},
			requests: []kargoapi.PromotionRequest{request("f.01", kargoapi.PromotionRequestPhaseSucceeded, nil, "t1")},
			expected: nil,
		},
		{
			name:     "Targets no Stage has promoted to are not healthy",
			targets:  []string{"t1", "t2"},
			stages:   []kargoapi.Stage{targetAware("fleet", "", ""), classic("classic")},
			expected: &kargoapi.TargetStats{Count: 2},
		},
		{
			name:    "healthy when the latest promotion from every Stage succeeded",
			targets: []string{"t1", "t2", "t3"},
			stages: []kargoapi.Stage{
				targetAware("a", "", "a.01"),
				targetAware("b", "", "b.01"),
			},
			requests: []kargoapi.PromotionRequest{
				request("a.01", kargoapi.PromotionRequestPhaseSucceeded,
					map[string]kargoapi.PromotionPhase{"t1": succeeded, "t2": succeeded, "t3": succeeded},
					"t1", "t2", "t3"),
				request("b.01", kargoapi.PromotionRequestPhaseErrored,
					map[string]kargoapi.PromotionPhase{"t1": succeeded, "t2": errored},
					"t1", "t2"),
			},
			// t1 succeeded from both; t2 errored from b; t3 succeeded from a alone.
			expected: &kargoapi.TargetStats{Count: 3, Health: kargoapi.HealthStats{Healthy: 2}},
		},
		{
			name:    "prefers the current request over the last",
			targets: []string{"t1"},
			stages:  []kargoapi.Stage{targetAware("a", "a.02", "a.01")},
			requests: []kargoapi.PromotionRequest{
				request("a.01", kargoapi.PromotionRequestPhaseSucceeded,
					map[string]kargoapi.PromotionPhase{"t1": succeeded}, "t1"),
				request("a.02", kargoapi.PromotionRequestPhaseRunning,
					map[string]kargoapi.PromotionPhase{"t1": running}, "t1"),
			},
			expected: &kargoapi.TargetStats{Count: 1},
		},
		{
			name:    "a Target without a recorded phase takes the request's own phase once it has ended",
			targets: []string{"t1", "t2", "t3"},
			stages: []kargoapi.Stage{
				targetAware("ok", "", "ok.01"),
				targetAware("bad", "", "bad.01"),
				targetAware("live", "live.01", ""),
			},
			requests: []kargoapi.PromotionRequest{
				request("ok.01", kargoapi.PromotionRequestPhaseSucceeded, nil, "t1"),
				request("bad.01", kargoapi.PromotionRequestPhaseErrored, nil, "t2"),
				request("live.01", kargoapi.PromotionRequestPhaseRunning, nil, "t3"),
			},
			expected: &kargoapi.TargetStats{Count: 3, Health: kargoapi.HealthStats{Healthy: 1}},
		},
		{
			name:    "a Stage whose latest request no longer exists is skipped",
			targets: []string{"t1"},
			stages: []kargoapi.Stage{
				targetAware("gone", "", "gone.01"),
				targetAware("here", "", "here.01"),
			},
			requests: []kargoapi.PromotionRequest{
				request("here.01", kargoapi.PromotionRequestPhaseSucceeded,
					map[string]kargoapi.PromotionPhase{"t1": succeeded}, "t1"),
			},
			expected: &kargoapi.TargetStats{Count: 1, Health: kargoapi.HealthStats{Healthy: 1}},
		},
		{
			name:    "a Target named by a request but no longer existing is not counted",
			targets: []string{"t1"},
			stages:  []kargoapi.Stage{targetAware("a", "", "a.01")},
			requests: []kargoapi.PromotionRequest{
				request("a.01", kargoapi.PromotionRequestPhaseSucceeded,
					map[string]kargoapi.PromotionPhase{"t1": succeeded, "deleted": succeeded},
					"t1", "deleted"),
			},
			expected: &kargoapi.TargetStats{Count: 1, Health: kargoapi.HealthStats{Healthy: 1}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			counts, err := fakeTargetCounter{targets: testCase.targets}.Count(
				t.Context(),
				testProject,
				healthyTargetNames(testCase.stages, testCase.requests),
			)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, targetStats(counts))
		})
	}
}
