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
	"github.com/akuity/kargo/pkg/database"
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
		store      *fakeStatsStore
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
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			store:  &fakeStatsStore{targetsErr: fmt.Errorf("something went wrong")},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "error listing Targets: something went wrong")
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
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			store:  &fakeStatsStore{requestsErr: fmt.Errorf("something went wrong")},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "error listing PromotionRequests: something went wrong")
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
			).Build(),
			store: &fakeStatsStore{
				targets: []database.Target{
					{Name: "t1", Labels: []byte(`{}`), Params: []byte(`{}`)},
					{Name: "t2", Labels: []byte(`{}`), Params: []byte(`{}`)},
				},
				requests: []database.PromotionRequestSnapshot{{
					PromotionRequest: database.PromotionRequest{
						Name:  "fleet.01",
						Phase: string(kargoapi.PromotionRequestPhaseErrored),
					},
					ProjectName: testProject,
					Stage:       "fleet",
					Freight:     "f",
					Targets: []database.PromotionRequestTargetRow{
						{Name: "t1", Ordinal: 0, Promotion: "p1", Phase: string(kargoapi.PromotionPhaseSucceeded)},
						{Name: "t2", Ordinal: 1, Promotion: "p2", Phase: string(kargoapi.PromotionPhaseErrored)},
					},
				}},
			},
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
		{
			name: "without a database there are no Target stats",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{Name: "fleet", Namespace: testProject},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{Steps: []kargoapi.PromotionStep{{}}},
						},
						Targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{{}}},
					},
				},
			).Build(),
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.Stats)
				require.Equal(t, int64(1), status.Stats.Stages.Count)
				require.Nil(t, status.Stats.Targets)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{client: tt.client}
			if tt.store != nil {
				r.store = tt.store
			}
			status, err := r.collectStats(t.Context(), tt.project)
			tt.assertions(t, status, err)
		})
	}
}

// fakeStatsStore is an in-memory projectStatsStore.
type fakeStatsStore struct {
	targets     []database.Target
	targetsErr  error
	requests    []database.PromotionRequestSnapshot
	requestsErr error
}

func (s *fakeStatsStore) ListTargets(context.Context, string) ([]database.Target, error) {
	return s.targets, s.targetsErr
}

func (s *fakeStatsStore) ListPromotionRequests(
	context.Context,
	string,
) ([]database.PromotionRequestSnapshot, error) {
	return s.requests, s.requestsErr
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
				// A classic Stage never has this, but even if it did, it must
				// not be consulted.
				LastPromotionRequest: &kargoapi.PromotionRequestReference{Name: "ignored"},
			},
		}
	}
	targets := func(names ...string) []kargoapi.Target {
		out := make([]kargoapi.Target, len(names))
		for i, name := range names {
			out[i].Name = name
		}
		return out
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
		targets  []kargoapi.Target
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
			targets:  targets("t1", "t2"),
			stages:   []kargoapi.Stage{targetAware("fleet", "", ""), classic("classic")},
			expected: &kargoapi.TargetStats{Count: 2},
		},
		{
			name:    "healthy when the latest promotion from every Stage succeeded",
			targets: targets("t1", "t2", "t3"),
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
			targets: targets("t1"),
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
			targets: targets("t1", "t2", "t3"),
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
			targets: targets("t1"),
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
			targets: targets("t1"),
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
			require.Equal(
				t,
				testCase.expected,
				collectTargetStats(testCase.targets, testCase.stages, testCase.requests),
			)
		})
	}
}
