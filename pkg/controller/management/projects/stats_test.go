package projects

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
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

// fakeRolloutReader serves a fixed set of rollouts as a Project's latest, or
// fails.
type fakeRolloutReader struct {
	rollouts []promotionrolloutstore.Rollout
	err      error
}

func (f fakeRolloutReader) ListLatest(
	_ context.Context,
	_ string,
	stage string,
) ([]promotionrolloutstore.Rollout, error) {
	if f.err != nil {
		return nil, f.err
	}
	var found []promotionrolloutstore.Rollout
	for _, rollout := range f.rollouts {
		if stage == "" || rollout.StageName == stage {
			found = append(found, rollout)
		}
	}
	return found, nil
}

// storedRollout builds a rollout as the store returns it: its row in the
// given phase and the named Targets.
func storedRollout(
	name, stage string,
	phase kargoapi.PromotionRolloutPhase,
	targets ...string,
) promotionrolloutstore.Rollout {
	rollout := promotionrolloutstore.Rollout{
		PromotionRolloutRow: database.PromotionRolloutRow{
			ID:          rolloutID(name),
			ProjectName: "fake-project",
			Name:        name,
			StageName:   stage,
			FreightName: "f",
			StatusPhase: string(phase),
		},
	}
	for _, target := range targets {
		rollout.Targets = append(rollout.Targets, database.ListPromotionRolloutTargetsRow{
			PromotionRolloutID: rollout.ID,
			TargetName:         target,
		})
	}
	return rollout
}

// rolloutID is the id storedRollout gives the named rollout, so that a
// rollout's children can name it.
func rolloutID(name string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(name))
}

// child builds a Promotion created by the named rollout to promote to the
// Target, in the given phase.
func child(name, rollout, target string, phase kargoapi.PromotionPhase) *kargoapi.Promotion {
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "fake-project",
			Annotations: map[string]string{
				kargoapi.AnnotationKeyPromotionRollout: rolloutID(rollout).String(),
			},
		},
		Spec:   kargoapi.PromotionSpec{Stage: "fleet", Target: target},
		Status: kargoapi.PromotionStatus{Phase: phase},
	}
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
		rollouts   RolloutReader
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
			client:   fake.NewClientBuilder().WithScheme(scheme).Build(),
			targets:  fakeTargetCounter{err: fmt.Errorf("something went wrong")},
			rollouts: fakeRolloutReader{},
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
			name: "error reading PromotionRollouts",
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
						Targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{{}}},
					},
				},
			).Build(),
			targets:  fakeTargetCounter{},
			rollouts: fakeRolloutReader{err: fmt.Errorf("something went wrong")},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				cond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, cond)
				require.Equal(t, metav1.ConditionFalse, cond.Status)
				require.Equal(t, "CollectingTargetStatsFailed", cond.Reason)
				require.Contains(t, cond.Message, "Failed to read promotion rollouts")
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
				},
				child("p1", "fleet.01", "t1", kargoapi.PromotionPhaseSucceeded),
				child("p2", "fleet.01", "t2", kargoapi.PromotionPhaseErrored),
			).Build(),
			targets: fakeTargetCounter{targets: []string{"t1", "t2"}},
			rollouts: fakeRolloutReader{rollouts: []promotionrolloutstore.Rollout{
				storedRollout("fleet.01", "fleet", kargoapi.PromotionRolloutPhaseErrored, "t1", "t2"),
			}},
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
			r := &reconciler{client: tt.client, targets: tt.targets, rollouts: tt.rollouts}
			status, err := r.collectStats(t.Context(), tt.project)
			tt.assertions(t, status, err)
		})
	}
}

func Test_targetStats(t *testing.T) {
	const testProject = "fake-project"

	targetAware := func(name string) kargoapi.Stage {
		stage := kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testProject},
			Spec: kargoapi.StageSpec{
				Targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{{}}},
			},
		}
		return stage
	}
	classic := func(name string) kargoapi.Stage {
		return kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testProject},
		}
	}
	// request builds a PromotionRollout of the Stage its name begins with,
	// naming the given Targets, with the given per-Target phases recorded in
	// status where provided.
	request := func(
		name string,
		phase kargoapi.PromotionRolloutPhase,
		phases map[string]kargoapi.PromotionPhase,
		names ...string,
	) kargoapi.PromotionRollout {
		req := kargoapi.PromotionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testProject},
			// A rollout is named after its Stage.
			Spec:   kargoapi.PromotionRolloutSpec{Stage: strings.SplitN(name, ".", 2)[0]},
			Status: kargoapi.PromotionRolloutStatus{Phase: phase},
		}
		for _, n := range names {
			req.Spec.Targets = append(req.Spec.Targets, kargoapi.PromotionRolloutTarget{Name: n})
			if p, ok := phases[n]; ok {
				req.Status.Targets = append(req.Status.Targets, kargoapi.PromotionRolloutTargetStatus{
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
		requests []kargoapi.PromotionRollout
		expected *kargoapi.TargetStats
	}{
		{
			name:     "no Targets",
			stages:   []kargoapi.Stage{targetAware("fleet")},
			requests: []kargoapi.PromotionRollout{request("f.01", kargoapi.PromotionRolloutPhaseSucceeded, nil, "t1")},
			expected: nil,
		},
		{
			name:     "Targets no Stage has promoted to are not healthy",
			targets:  []string{"t1", "t2"},
			stages:   []kargoapi.Stage{targetAware("fleet"), classic("classic")},
			expected: &kargoapi.TargetStats{Count: 2},
		},
		{
			name:    "healthy when the latest promotion from every Stage succeeded",
			targets: []string{"t1", "t2", "t3"},
			stages: []kargoapi.Stage{
				targetAware("a"),
				targetAware("b"),
			},
			requests: []kargoapi.PromotionRollout{
				request("a.01", kargoapi.PromotionRolloutPhaseSucceeded,
					map[string]kargoapi.PromotionPhase{"t1": succeeded, "t2": succeeded, "t3": succeeded},
					"t1", "t2", "t3"),
				request("b.01", kargoapi.PromotionRolloutPhaseErrored,
					map[string]kargoapi.PromotionPhase{"t1": succeeded, "t2": errored},
					"t1", "t2"),
			},
			// t1 succeeded from both; t2 errored from b; t3 succeeded from a alone.
			expected: &kargoapi.TargetStats{Count: 3, Health: kargoapi.HealthStats{Healthy: 2}},
		},
		{
			name:    "prefers the rollout in progress over the one that finished",
			targets: []string{"t1"},
			stages:  []kargoapi.Stage{targetAware("a")},
			requests: []kargoapi.PromotionRollout{
				request("a.01", kargoapi.PromotionRolloutPhaseSucceeded,
					map[string]kargoapi.PromotionPhase{"t1": succeeded}, "t1"),
				request("a.02", kargoapi.PromotionRolloutPhaseRunning,
					map[string]kargoapi.PromotionPhase{"t1": running}, "t1"),
			},
			expected: &kargoapi.TargetStats{Count: 1},
		},
		{
			name:    "a Target without a recorded phase takes the request's own phase once it has ended",
			targets: []string{"t1", "t2", "t3"},
			stages: []kargoapi.Stage{
				targetAware("ok"),
				targetAware("bad"),
				targetAware("live"),
			},
			requests: []kargoapi.PromotionRollout{
				request("ok.01", kargoapi.PromotionRolloutPhaseSucceeded, nil, "t1"),
				request("bad.01", kargoapi.PromotionRolloutPhaseErrored, nil, "t2"),
				request("live.01", kargoapi.PromotionRolloutPhaseRunning, nil, "t3"),
			},
			expected: &kargoapi.TargetStats{Count: 3, Health: kargoapi.HealthStats{Healthy: 1}},
		},
		{
			name:    "a Stage with no rollouts is skipped",
			targets: []string{"t1"},
			stages: []kargoapi.Stage{
				targetAware("gone"),
				targetAware("here"),
			},
			requests: []kargoapi.PromotionRollout{
				request("here.01", kargoapi.PromotionRolloutPhaseSucceeded,
					map[string]kargoapi.PromotionPhase{"t1": succeeded}, "t1"),
			},
			expected: &kargoapi.TargetStats{Count: 1, Health: kargoapi.HealthStats{Healthy: 1}},
		},
		{
			name:    "a Target named by a request but no longer existing is not counted",
			targets: []string{"t1"},
			stages:  []kargoapi.Stage{targetAware("a")},
			requests: []kargoapi.PromotionRollout{
				request("a.01", kargoapi.PromotionRolloutPhaseSucceeded,
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
