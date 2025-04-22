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
	"github.com/akuity/kargo/internal/conditions"
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
					Status: kargoapi.StageStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionFalse,
						}},
					},
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stage3",
						Namespace: testProject,
					},
					// No health condition == unknown
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stage4",
						Namespace: testProject,
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
					Status: kargoapi.StageStatus{
						Conditions: []metav1.Condition{{
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionUnknown,
						}},
					},
				},
			).Build(),
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				require.Nil(t, conditions.Get(&status, kargoapi.ConditionTypeHealthy))
				stats := status.Stats
				require.Equal(t, int64(1), stats.Warehouses.Health.Healthy)
				require.Equal(t, int64(1), stats.Warehouses.Health.Unhealthy)
				require.Equal(t, int64(3), stats.Warehouses.Health.Unknown)
				require.Equal(t, int64(1), stats.Stages.Health.Healthy)
				require.Equal(t, int64(1), stats.Stages.Health.Unhealthy)
				require.Equal(t, int64(3), stats.Stages.Health.Unknown)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{client: tt.client}
			status, err := r.collectStats(context.Background(), tt.project)
			tt.assertions(t, status, err)
		})
	}
}
