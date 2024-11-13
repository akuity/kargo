package stages

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
	rolloutsapi "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/directives"
	"github.com/akuity/kargo/internal/indexer"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
)

func Test_regularStagesReconciler_syncPromotions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	now := time.Now()
	hourAgo := now.Add(-time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, kargoapi.StageStatus, bool, error)
	}{
		{
			name: "list promotions error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("list error")
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.ErrorContains(t, err, "failed to list Promotions")
				assert.False(t, hasPendingPromotions)

				assert.Len(t, status.Conditions, 1)
				assert.Equal(t, kargoapi.ConditionTypePromoting, status.Conditions[0].Type)
				assert.Equal(t, metav1.ConditionUnknown, status.Conditions[0].Status)
				require.Contains(t, status.Conditions[0].Message, "failed to list Promotions")
			},
		},
		{
			name: "no promotions",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "old-promotion",
					},
					Conditions: []metav1.Condition{
						{
							Type: kargoapi.ConditionTypePromoting,
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)
				assert.Empty(t, status.Conditions)
			},
		},
		{
			name: "successful promotion updates freight history",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{
						{
							Type: kargoapi.ConditionTypePromoting,
						},
					},
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "successful-promotion",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "successful-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: now},
						FreightCollection: &kargoapi.FreightCollection{
							ID: "test-collection-id",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {
									Name: "test-freight-1",
								},
								"warehouse-2": {
									Name: "test-freight-2",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)

				assert.False(t, hasPendingPromotions)

				assert.Nil(t, status.CurrentPromotion)

				require.NotNil(t, status.LastPromotion)
				assert.Equal(t, "successful-promotion", status.LastPromotion.Name)

				// Verify freight history
				require.Len(t, status.FreightHistory, 1)
				assert.Equal(t, &kargoapi.FreightCollection{
					ID: "test-collection-id",
					Freight: map[string]kargoapi.FreightReference{
						"warehouse-1": {
							Name: "test-freight-1",
						},
						"warehouse-2": {
							Name: "test-freight-2",
						},
					},
				}, status.FreightHistory[0])

				// Verify conditions are set correctly
				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, "WaitingForHealthCheck", healthyCond.Reason)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionUnknown, verifiedCond.Status)
				assert.Equal(t, "WaitingForVerification", verifiedCond.Reason)
			},
		},
		{
			name: "active promotion updates status",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "active-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
						Freight: &kargoapi.FreightReference{
							Name: "test-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)

				assert.True(t, hasPendingPromotions)

				assert.Equal(t, &kargoapi.PromotionReference{
					Name: "active-promotion",
					Freight: &kargoapi.FreightReference{
						Name: "test-freight",
					},
				}, status.CurrentPromotion)

				// Verify promoting condition is set
				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
				assert.Equal(t, "ActivePromotion", promotingCond.Reason)
			},
		},
		{
			name: "blocks new promotion when current freight has running verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {
									Name: "current-freight",
								},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseRunning,
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)
				assert.Empty(t, status.Conditions)
			},
		},
		{
			name: "waits for running verification even when health is unhealthy",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateUnhealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{Phase: kargoapi.VerificationPhaseRunning},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				// Should not have current promotion since verification is running
				assert.Nil(t, status.CurrentPromotion)
				// Should not have promoting condition
				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)
			},
		},
		{
			name: "allows promotion after verification completes when health is unhealthy",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateUnhealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase:      kargoapi.VerificationPhaseSuccessful,
									FinishTime: &metav1.Time{Time: time.Now()},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				// Should allow promotion after verification is complete
				require.NotNil(t, status.CurrentPromotion)
				assert.Equal(t, "pending-promotion", status.CurrentPromotion.Name)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
			},
		},
		{
			name: "waits for verification even with no health check",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: nil, // No health check performed
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{Phase: kargoapi.VerificationPhaseRunning},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)
			},
		},
		{
			name: "allows promotion when unhealthy and no verification exists",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateUnhealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
							},
							// Empty verification history
							VerificationHistory: []kargoapi.VerificationInfo{},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
						Freight: &kargoapi.FreightReference{
							Name: "new-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				// Should allow promotion since there's no verification to wait for
				require.NotNil(t, status.CurrentPromotion)
				assert.Equal(t, "pending-promotion", status.CurrentPromotion.Name)
				assert.Equal(t, "new-freight", status.CurrentPromotion.Freight.Name)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
				assert.Equal(t, "ActivePromotion", promotingCond.Reason)
				assert.Contains(t, promotingCond.Message, "Pending")
			},
		},
		{
			name: "skips older promotions after last promotion",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Name: "promotion-2",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "promotion-1",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: hourAgo},
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "promotion-2",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: now},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)

				assert.Equal(t, "promotion-2", status.LastPromotion.Name)
				assert.Empty(t, status.FreightHistory)
			},
		},
		{
			name: "processes failed promotions without updating freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "failed-promotion",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "failed-promotion",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseFailed,
						FinishedAt: &metav1.Time{Time: now},
						FreightCollection: &kargoapi.FreightCollection{
							ID: "failed-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "failed-freight"},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)

				require.NotNil(t, status.LastPromotion)
				assert.Equal(t, "failed-promotion", status.LastPromotion.Name)
				assert.Empty(t, status.FreightHistory)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)
			},
		},
		{
			name: "handles promotion phase transition",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "transitioning-promotion",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "transitioning-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseRunning,
						Freight: &kargoapi.FreightReference{
							Name: "test-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				assert.Equal(t, "transitioning-promotion", status.CurrentPromotion.Name)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
				assert.Equal(t, "ActivePromotion", promotingCond.Reason)
				assert.Contains(t, promotingCond.Message, "Running")
			},
		},
		{
			name: "highest priority promotion has already been processed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Name: "terminal-promotion",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "terminal-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: time.Now()},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithIndex(
					&kargoapi.Promotion{},
					indexer.PromotionsByStageField,
					indexer.PromotionsByStage,
				).
				WithStatusSubresource(&kargoapi.Stage{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &RegularStagesReconciler{
				client: c,
			}

			status, requeue, err := r.syncPromotions(context.Background(), tt.stage)
			tt.assertions(t, status, requeue, err)
		})
	}
}

func Test_regularStagesReconciler_assessHealth(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name          string
		stage         *kargoapi.Stage
		checkHealthFn func(context.Context, directives.HealthCheckContext, []directives.HealthCheckStep) kargoapi.Health
		assertions    func(*testing.T, kargoapi.StageStatus)
	}{
		{
			name: "no last promotion",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: nil,
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				assert.Nil(t, status.Health)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, "NoFreight", healthyCond.Reason)
				assert.Equal(t, "Stage has no current Freight", healthyCond.Message)
			},
		},
		{
			name: "no health checks",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							HealthChecks: nil,
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				assert.Nil(t, status.Health)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, "NoHealthChecks", healthyCond.Reason)
				assert.Equal(t, "Stage has no health checks to perform", healthyCond.Message)
			},
		},
		{
			name: "healthy state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							HealthChecks: []kargoapi.HealthCheckStep{
								{
									Uses: "test-check",
								},
							},
						},
					},
				},
			},
			checkHealthFn: func(
				context.Context,
				directives.HealthCheckContext,
				[]directives.HealthCheckStep,
			) kargoapi.Health {
				return kargoapi.Health{Status: kargoapi.HealthStateHealthy}
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				require.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateHealthy, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionTrue, healthyCond.Status)
				assert.Equal(t, string(kargoapi.HealthStateHealthy), healthyCond.Reason)
				assert.Equal(t, "Stage is healthy", healthyCond.Message)
			},
		},
		{
			name: "unhealthy state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							HealthChecks: []kargoapi.HealthCheckStep{
								{
									Uses: "test-check",
								},
							},
						},
					},
				},
			},
			checkHealthFn: func(
				context.Context,
				directives.HealthCheckContext,
				[]directives.HealthCheckStep,
			) kargoapi.Health {
				return kargoapi.Health{Status: kargoapi.HealthStateUnhealthy}
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				require.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateUnhealthy, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionFalse, healthyCond.Status)
				assert.Equal(t, string(kargoapi.HealthStateUnhealthy), healthyCond.Reason)
				assert.Equal(t, "Stage is unhealthy", healthyCond.Message)
			},
		},
		{
			name: "not applicable state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{},
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							HealthChecks: []kargoapi.HealthCheckStep{
								{
									Uses: "test-check",
								},
							},
						},
					},
				},
			},
			checkHealthFn: func(
				context.Context,
				directives.HealthCheckContext,
				[]directives.HealthCheckStep,
			) kargoapi.Health {
				return kargoapi.Health{Status: kargoapi.HealthStateNotApplicable}
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				require.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateNotApplicable, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				assert.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
			},
		},
		{
			name: "unknown state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							HealthChecks: []kargoapi.HealthCheckStep{
								{
									Uses: "test-check",
								},
							},
						},
					},
				},
			},
			checkHealthFn: func(
				context.Context,
				directives.HealthCheckContext,
				[]directives.HealthCheckStep,
			) kargoapi.Health {
				return kargoapi.Health{Status: kargoapi.HealthStateUnknown}
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				require.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateUnknown, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, string(kargoapi.HealthStateUnknown), healthyCond.Reason)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			r := &RegularStagesReconciler{
				client: c,
				directivesEngine: &directives.FakeEngine{
					CheckHealthFn: tt.checkHealthFn,
				},
			}

			status := r.assessHealth(context.Background(), tt.stage)
			tt.assertions(t, status)
		})
	}
}

func Test_regularStagesReconciler_verifyStageFreight(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	startTime := time.Now()
	endTime := startTime.Add(5 * time.Minute)
	fixedEndTime := func() time.Time { return endTime }

	tests := []struct {
		name             string
		stage            *kargoapi.Stage
		objects          []client.Object
		assertions       func(*testing.T, client.Client, *fakeevent.EventRecorder, kargoapi.StageStatus, error)
		rolloutsDisabled bool
	}{
		{
			name: "no current freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: nil,
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 0)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionUnknown, verifiedCond.Status)
				assert.Equal(t, "NoFreight", verifiedCond.Reason)
			},
		},
		{
			name: "skips verification when promotion is running",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "running-promotion",
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 0)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				assert.Nil(t, verifiedCond)
			},
		},
		{
			name: "verifies without verification config",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: nil,
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse":   {Name: "test-freight"},
								"warehouse-2": {Name: "test-freight-2"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight-2",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				require.Len(t, recorder.Events, 2)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, lastVerification.Phase)
				assert.Equal(t, metav1.NewTime(startTime), *lastVerification.StartTime)
				assert.Equal(t, metav1.NewTime(endTime), *lastVerification.FinishTime)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionTrue, verifiedCond.Status)
				assert.Equal(t, "Verified", verifiedCond.Reason)
				assert.Equal(t, "Freight has been verified", verifiedCond.Message)
			},
		},
		{
			name: "skips verification with nil health status",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: nil,
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				_ *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)
				assert.Empty(t, curFreight.VerificationHistory)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.Nil(t, verifiedCond)
			},
		},
		{
			name: "skips verification when unhealthy",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateUnhealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				_ *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)
				assert.Empty(t, curFreight.VerificationHistory)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.Nil(t, verifiedCond)
			},
		},
		{
			name:             "error when rollouts integration is disabled",
			rolloutsDisabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseError, lastVerification.Phase)
				assert.Contains(t, lastVerification.Message, "Rollouts integration is disabled")

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionFalse, verifiedCond.Status)
				assert.Equal(t, "VerificationError", verifiedCond.Reason)
				assert.Contains(t, verifiedCond.Message, "Rollouts integration is disabled")
			},
		},
		{
			name: "handles verification abort request",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "test-verification-id",
					},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "test-verification-id",
									Phase: kargoapi.VerificationPhaseRunning,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "test-analysis-run",
										Namespace: "fake-project",
									},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis-run",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				c client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, lastVerification.Phase)
				assert.Contains(t, lastVerification.Message, "aborted by user")

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionFalse, verifiedCond.Status)
				assert.Equal(t, "VerificationFailed", verifiedCond.Reason)
				assert.Contains(t, verifiedCond.Message, "aborted by user")

				// Verify AnalysisRun was patched to terminate
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Name:      "test-analysis-run",
					Namespace: "fake-project",
				}, ar))
				assert.True(t, ar.Spec.Terminate)
			},
		},
		{
			name: "handles re-verification request",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: `{"id":"test-verification-id","actor":"test-user"}`,
					},
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					LastPromotion: &kargoapi.PromotionReference{
						Name: "last-promotion",
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "test-verification-id",
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				c client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 0)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhasePending, lastVerification.Phase)
				assert.NotEmpty(t, lastVerification.ID)
				assert.Equal(t, "test-user", lastVerification.Actor)

				// As we have a successful (previous) verification, we should have a verified condition
				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionTrue, verifiedCond.Status)
				assert.Equal(t, "Verified", verifiedCond.Reason)
				assert.Equal(t, "Freight has been verified", verifiedCond.Message)

				// Verify new AnalysisRun was created
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Name:      lastVerification.AnalysisRun.Name,
					Namespace: lastVerification.AnalysisRun.Namespace,
				}, ar))
			},
		},
		{
			name: "continues existing non-terminal verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:        "test-verification-id",
									Phase:     kargoapi.VerificationPhaseRunning,
									StartTime: &metav1.Time{Time: startTime},
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "test-analysis-run",
										Namespace: "fake-project",
									},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis-run",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Running",
						Message: "Analysis is running",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 0)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseRunning, lastVerification.Phase)
				assert.Equal(t, "test-verification-id", lastVerification.ID)
				assert.Equal(t, "test-analysis-run", lastVerification.AnalysisRun.Name)
				assert.Equal(t, "Running", lastVerification.AnalysisRun.Phase)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionUnknown, verifiedCond.Status)
				assert.Equal(t, "VerificationRunning", verifiedCond.Reason)
				assert.Equal(t, "Freight is currently being verified", verifiedCond.Message)
			},
		},
		{
			name: "handles error getting AnalysisRun for freight verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "test-verification-id",
									Phase: kargoapi.VerificationPhaseRunning,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "missing-analysis-run",
										Namespace: "fake-project",
									},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.True(t, apierrors.IsNotFound(err))

				assert.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseError, lastVerification.Phase)
				assert.Contains(t, lastVerification.Message, "error getting AnalysisRun")

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionFalse, verifiedCond.Status)
				assert.Equal(t, "VerificationError", verifiedCond.Reason)
				assert.Contains(t, verifiedCond.Message, "error getting AnalysisRun")
			},
		},
		{
			name: "uses existing analysis run for freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "test-freight-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Successful",
						Message: "Analysis completed successfully",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, lastVerification.Phase)
				assert.Equal(t, "existing-analysis", lastVerification.AnalysisRun.Name)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionTrue, verifiedCond.Status)
				assert.Equal(t, "Verified", verifiedCond.Reason)
				assert.Equal(t, "Freight has been verified", verifiedCond.Message)
			},
		},
		{
			name: "handles multiple verification histories with re-verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: `{"id":"second-verification","actor":"test-user"}`,
					},
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:        "second-verification",
									Phase:     kargoapi.VerificationPhaseSuccessful,
									StartTime: &metav1.Time{Time: startTime.Add(-time.Hour)},
								},
								{
									ID:        "first-verification",
									Phase:     kargoapi.VerificationPhaseSuccessful,
									StartTime: &metav1.Time{Time: startTime.Add(-2 * time.Hour)},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				_ *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)
				require.Len(t, curFreight.VerificationHistory, 3)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhasePending, lastVerification.Phase)
				assert.NotEmpty(t, lastVerification.ID)
				assert.Equal(t, "test-user", lastVerification.Actor)

				// Should be true as we have a successful verification
				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionTrue, verifiedCond.Status)
				assert.Equal(t, "Verified", verifiedCond.Reason)
				assert.Equal(t, "Freight has been verified", verifiedCond.Message)

				// Verify the previous verifications are preserved
				assert.Equal(t, "second-verification", curFreight.VerificationHistory[1].ID)
				assert.Equal(t, "first-verification", curFreight.VerificationHistory[2].ID)
			},
		},
		{
			name: "handles terminal analysis run state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "test-verification-id",
									Phase: kargoapi.VerificationPhaseRunning,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "test-analysis-run",
										Namespace: "fake-project",
									},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis-run",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Failed",
						Message: "Analysis failed due to metric error",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				require.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, lastVerification.Phase)
				assert.Equal(t, "test-analysis-run", lastVerification.AnalysisRun.Name)
				assert.Equal(t, "Failed", lastVerification.AnalysisRun.Phase)
				assert.Contains(t, lastVerification.Message, "Analysis failed")

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionFalse, verifiedCond.Status)
				assert.Equal(t, "VerificationFailed", verifiedCond.Reason)
				assert.Contains(t, verifiedCond.Message, "Analysis failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}).
				Build()

			recorder := fakeevent.NewEventRecorder(10)

			r := &RegularStagesReconciler{
				client: c,
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				},
				eventRecorder: recorder,
			}

			status, err := r.verifyStageFreight(context.Background(), tt.stage, startTime, fixedEndTime)
			tt.assertions(t, c, recorder, status, err)
		})
	}
}

func Test_regularStagesReconciler_verifyFreightForStage(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		objects    []client.Object
		assertions func(*testing.T, client.Client, kargoapi.StageStatus, error)
	}{
		{
			name: "skips verification when unhealthy",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateUnhealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				// Status should remain unchanged
				assert.Equal(t, kargoapi.HealthStateUnhealthy, status.Health.Status)
			},
		},
		{
			name: "skips verification when no current freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "skips verification when non-terminal verification exists",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseRunning,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "skips verification when last verification is not successful",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseFailed,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "handles freight not found",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "missing-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "error getting Freight")
			},
		},
		{
			name: "marks freight as verified when not already verified",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-freight",
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)

				// Check if freight was properly marked as verified
				freight := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), client.ObjectKey{
					Namespace: "fake-project",
					Name:      "test-freight",
				}, freight))

				verifiedStage, ok := freight.Status.VerifiedIn["test-stage"]
				require.True(t, ok)
				assert.Equal(t, kargoapi.VerifiedStage{}, verifiedStage)
			},
		},
		{
			name: "skips already verified freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-freight",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)

				// Verify no changes were made to the freight
				freight := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), client.ObjectKey{
					Namespace: "fake-project",
					Name:      "test-freight",
				}, freight))
				assert.Len(t, freight.Status.VerifiedIn, 1)
			},
		},
		{
			name: "handles multiple freight references",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse1": {Name: "freight-1"},
								"warehouse2": {Name: "freight-2"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "freight-2",
					},
					Status: kargoapi.FreightStatus{},
				},
			},
			assertions: func(t *testing.T, c client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)

				// Check both freight objects were marked as verified
				for _, name := range []string{"freight-1", "freight-2"} {
					freight := &kargoapi.Freight{}
					require.NoError(t, c.Get(context.Background(), client.ObjectKey{
						Namespace: "fake-project",
						Name:      name,
					}, freight))

					verifiedStage, ok := freight.Status.VerifiedIn["test-stage"]
					require.True(t, ok, "freight %s should be verified", name)
					assert.Equal(t, kargoapi.VerifiedStage{}, verifiedStage)
				}
			},
		},
		{
			name: "handles patch error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       "fake-project",
						Name:            "test-freight",
						ResourceVersion: "invalid", // This will cause patch to fail
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "error marking Freight")
			},
		},
		{
			name: "empty verification history skips verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "nil health status skips verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: nil,
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				Build()

			r := &RegularStagesReconciler{
				client:           c,
				directivesEngine: &directives.FakeEngine{},
			}

			status, err := r.verifyFreightForStage(context.Background(), tt.stage)
			tt.assertions(t, c, status, err)
		})
	}
}

func Test_regularStagesReconciler_recordFreightVerificationEvent(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := metav1.Now()
	startTime := metav1.NewTime(now.Add(-1 * time.Hour))
	finishTime := metav1.NewTime(now.Add(-30 * time.Minute))

	baseStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stage",
			Namespace: "test-ns",
		},
	}

	baseFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-ns",
			CreationTimestamp: now,
		},
		Alias: "test-alias",
	}

	baseFreightRef := kargoapi.FreightReference{
		Name: "test-freight",
	}

	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		freightRef kargoapi.FreightReference
		vi         *kargoapi.VerificationInfo
		objects    []client.Object
		assertions func(*testing.T, *fakeevent.EventRecorder)
	}{
		{
			name:       "successful verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:      kargoapi.VerificationPhaseSuccessful,
				StartTime:  &startTime,
				FinishTime: &finishTime,
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, corev1.EventTypeNormal, event.EventType)
				assert.Equal(t, kargoapi.EventReasonFreightVerificationSucceeded, event.Reason)
				assert.Equal(t, "Freight verification succeeded", event.Message)

				assert.Equal(t, baseStage.Name, event.Annotations[kargoapi.AnnotationKeyEventStageName])
				assert.Equal(t, baseFreight.Alias, event.Annotations[kargoapi.AnnotationKeyEventFreightAlias])
				assert.Equal(
					t,
					startTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				assert.Equal(
					t,
					finishTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)
			},
		},
		{
			name:       "failed verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   kargoapi.VerificationPhaseFailed,
				Message: "verification failed due to metrics",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, kargoapi.EventReasonFreightVerificationFailed, event.Reason)
				assert.Equal(t, "verification failed due to metrics", event.Message)
			},
		},
		{
			name:       "verification with analysis run and promotion",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseSuccessful,
				AnalysisRun: &kargoapi.AnalysisRunReference{
					Name:      "test-analysis",
					Namespace: "test-ns",
				},
			},
			objects: []client.Object{
				baseFreight,
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "test-ns",
						Labels: map[string]string{
							kargoapi.PromotionLabelKey: "test-promotion",
						},
					},
				},
			},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, "test-analysis", event.Annotations[kargoapi.AnnotationKeyEventAnalysisRunName])
				assert.Equal(t, "test-promotion", event.Annotations[kargoapi.AnnotationKeyEventPromotionName])
			},
		},
		{
			name:       "verification with manual actor override",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseSuccessful,
				Actor: "manual-user",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, "manual-user", event.Annotations[kargoapi.AnnotationKeyEventActor])
			},
		},
		{
			name:       "freight not found",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseSuccessful,
			},
			objects: []client.Object{
				// Freight does not exist
			},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				// No events should be recorded
				assert.Len(t, recorder.Events, 0)
			},
		},
		{
			name:       "analysis run not found",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseSuccessful,
				AnalysisRun: &kargoapi.AnalysisRunReference{
					Name:      "missing-analysis",
					Namespace: "test-ns",
				},
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, "missing-analysis", event.Annotations[kargoapi.AnnotationKeyEventAnalysisRunName])
				// Should still record event even though analysis run wasn't found
				assert.NotContains(t, event.Annotations, kargoapi.AnnotationKeyEventPromotionName)
			},
		},
		{
			name:       "errored verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   kargoapi.VerificationPhaseError,
				Message: "internal error occurred",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, kargoapi.EventReasonFreightVerificationErrored, event.Reason)
				assert.Equal(t, "internal error occurred", event.Message)
			},
		},
		{
			name:       "aborted verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   kargoapi.VerificationPhaseAborted,
				Message: "verification was canceled",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, kargoapi.EventReasonFreightVerificationAborted, event.Reason)
				assert.Equal(t, "verification was canceled", event.Message)
			},
		},
		{
			name:       "inconclusive verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   kargoapi.VerificationPhaseInconclusive,
				Message: "results were inconclusive",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, kargoapi.EventReasonFreightVerificationInconclusive, event.Reason)
				assert.Equal(t, "results were inconclusive", event.Message)
			},
		},
		{
			name:       "unknown phase",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   "invalid-phase",
				Message: "custom message",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, kargoapi.EventReasonFreightVerificationUnknown, event.Reason)
				assert.Equal(t, "custom message", event.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			recorder := fakeevent.NewEventRecorder(10)

			r := &RegularStagesReconciler{
				client:        c,
				eventRecorder: recorder,
			}

			r.recordFreightVerificationEvent(tt.stage, tt.freightRef, tt.vi)
			tt.assertions(t, recorder)
		})
	}
}

func Test_regularStagesReconciler_startVerification(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()

	tests := []struct {
		name             string
		stage            *kargoapi.Stage
		freightCol       kargoapi.FreightCollection
		req              *kargoapi.VerificationRequest
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, client.Client, *kargoapi.VerificationInfo, error)
	}{
		{
			name: "rollouts integration disabled",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "Rollouts integration is disabled")
				assert.Equal(t, now, vi.StartTime.Time)
				assert.NotNil(t, vi.FinishTime)
			},
		},
		{
			name: "finds existing analysis run",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "test-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Successful",
						Message: "Analysis completed successfully",
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, vi.Phase)
				assert.Equal(t, "existing-analysis", vi.AnalysisRun.Name)
			},
		},
		{
			name: "creates new analysis run",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, kargoapi.VerificationPhasePending, vi.Phase)
				assert.NotNil(t, vi.AnalysisRun)

				// Verify analysis run was created
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))
			},
		},
		{
			name: "handles reverification with control plane actor",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Name: "test-promotion",
					},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID: "prev-verification",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-freight",
					},
				},
			},
			req: &kargoapi.VerificationRequest{
				ID:           "prev-verification",
				Actor:        "test-user",
				ControlPlane: true,
			},
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, "test-user", vi.Actor)

				// Verify promotion label was added
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))
				assert.Equal(t, "test-promotion", ar.Labels[kargoapi.PromotionLabelKey])
			},
		},
		{
			name: "handles analysis run build error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			objects: []client.Object{
				// Missing Freight object for owner reference
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "error building AnalysisRun")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}, &rolloutsapi.AnalysisRun{}).
				Build()

			r := &RegularStagesReconciler{
				client: c,
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled:   !tt.rolloutsDisabled,
					RolloutsControllerInstanceID: "test-instance",
				},
			}

			vi, err := r.startVerification(context.Background(), tt.stage, tt.freightCol, tt.req, now)
			tt.assertions(t, c, vi, err)
		})
	}
}

func Test_regularStagesReconciler_getVerificationResult(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()
	fiveMinutesLater := now.Add(5 * time.Minute)

	tests := []struct {
		name             string
		freight          kargoapi.FreightCollection
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, *kargoapi.VerificationInfo, error)
	}{
		{
			name: "error when no current verification info",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.ErrorContains(t, err, "no current verification info")
				assert.Nil(t, vi)
			},
		},
		{
			name: "error when no analysis run reference",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.ErrorContains(t, err, "no AnalysisRun reference")
				assert.Nil(t, vi)
			},
		},
		{
			name: "error when rollouts integration disabled",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Contains(t, vi.Message, "Rollouts integration is disabled")
			},
		},
		{
			name: "error when analysis run not found",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "missing-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.True(t, apierrors.IsNotFound(err))

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Contains(t, vi.Message, "error getting AnalysisRun")
				assert.NotNil(t, vi.AnalysisRun)
				assert.Equal(t, "missing-analysis", vi.AnalysisRun.Name)
			},
		},
		{
			name: "preserves actor in verification info",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						Actor:     "test-user",
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Running",
						Message: "Analysis in progress",
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Equal(t, "test-user", vi.Actor)
				assert.Equal(t, kargoapi.VerificationPhaseRunning, vi.Phase)
			},
		},
		{
			name: "handles successful analysis run",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   rolloutsapi.AnalysisPhaseSuccessful,
						Message: "Analysis completed successfully",
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: fiveMinutesLater},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Equal(t, "Analysis completed successfully", vi.Message)
				assert.NotNil(t, vi.FinishTime)
			},
		},
		{
			name: "handles failed analysis run",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Failed",
						Message: "Something went wrong",
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: fiveMinutesLater},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Equal(t, "Something went wrong", vi.Message)
				assert.NotNil(t, vi.FinishTime)
				assert.Equal(t, string(rolloutsapi.AnalysisPhaseFailed), vi.AnalysisRun.Phase)
			},
		},
		{
			name: "handles error analysis run",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   rolloutsapi.AnalysisPhaseError,
						Message: "Something went wrong",
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: fiveMinutesLater},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Equal(t, "Something went wrong", vi.Message)
				assert.NotNil(t, vi.FinishTime)
				assert.Equal(t, string(rolloutsapi.AnalysisPhaseError), vi.AnalysisRun.Phase)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}, &rolloutsapi.AnalysisRun{}).
				Build()

			r := &RegularStagesReconciler{
				client: c,
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				},
			}

			vi, err := r.getVerificationResult(context.Background(), tt.freight)
			tt.assertions(t, vi, err)
		})
	}
}

func Test_regularStagesReconciler_abortVerification(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()

	tests := []struct {
		name             string
		freightCol       kargoapi.FreightCollection
		req              *kargoapi.VerificationRequest
		objects          []client.Object
		rolloutsDisabled bool
		interceptor      interceptor.Funcs
		assertions       func(*testing.T, client.Client, *kargoapi.VerificationInfo, error)
	}{
		{
			name: "error when no current verification info",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.ErrorContains(t, err, "no current verification info")
				assert.Nil(t, vi)
			},
		},
		{
			name: "error when no analysis run reference",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.ErrorContains(t, err, "no AnalysisRun reference")
				assert.Nil(t, vi)
			},
		},
		{
			name: "returns current verification if already terminal",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseSuccessful,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
			},
		},
		{
			name: "error when rollouts integration disabled",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "Rollouts integration is disabled")
				assert.Equal(t, "test-verification", vi.ID)
				assert.NotNil(t, vi.StartTime)
				assert.NotNil(t, vi.FinishTime)
			},
		},
		{
			name: "handles patch error",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				Patch: func(
					context.Context,
					client.WithWatch,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err) // Error is captured in verification info
				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "error terminating AnalysisRun")
			},
		},
		{
			name: "successfully aborts verification",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Spec: rolloutsapi.AnalysisRunSpec{
						Metrics: []rolloutsapi.Metric{
							{Name: "test-metric"},
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Running",
						Message: "Analysis in progress",
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, vi.Phase)
				assert.Equal(t, "Verification aborted by user", vi.Message)
				assert.Equal(t, "test-verification", vi.ID)
				assert.NotNil(t, vi.StartTime)
				assert.NotNil(t, vi.FinishTime)
				assert.Equal(t, "test-analysis", vi.AnalysisRun.Name)

				// Verify analysis run was patched with terminate = true
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "fake-project",
					Name:      "test-analysis",
				}, ar))
				assert.True(t, ar.Spec.Terminate)
			},
		},
		{
			name: "handles already terminated analysis run",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Spec: rolloutsapi.AnalysisRunSpec{
						Terminate: true,
						Metrics: []rolloutsapi.Metric{
							{Name: "test-metric"},
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Successful",
						Message: "Analysis completed",
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, vi.Phase)
				assert.Equal(t, "Verification aborted by user", vi.Message)
				assert.Equal(t, "test-verification", vi.ID)
				assert.NotNil(t, vi.StartTime)
				assert.NotNil(t, vi.FinishTime)
				assert.Equal(t, "test-analysis", vi.AnalysisRun.Name)
			},
		},
		{
			name: "sets actor in verification info",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			req: &kargoapi.VerificationRequest{
				ID:    "test-verification",
				Actor: "test-user",
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, "test-user", vi.Actor)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, vi.Phase)
			},
		},
		{
			name: "handles analysis run not found",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "missing-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "error terminating AnalysisRun")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}, &rolloutsapi.AnalysisRun{})

			if tt.interceptor.Patch != nil {
				builder = builder.WithInterceptorFuncs(tt.interceptor)
			}

			c := builder.Build()

			r := &RegularStagesReconciler{
				client: c,
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				},
			}

			vi, err := r.abortVerification(context.Background(), tt.freightCol, tt.req)
			tt.assertions(t, c, vi, err)
		})
	}
}

func Test_regularStagesReconciler_findExistingAnalysisRun(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()
	hourAgo := now.Add(-time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	tests := []struct {
		name         string
		stage        types.NamespacedName
		freightColID string
		objects      []client.Object
		interceptor  interceptor.Funcs
		assertions   func(*testing.T, *rolloutsapi.AnalysisRun, error)
	}{
		{
			name: "no analysis runs found",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)
				assert.Nil(t, ar)
			},
		},
		{
			name: "handles list error",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			interceptor: interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return fmt.Errorf("list error")
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.ErrorContains(t, err, "error listing AnalysisRuns")
				assert.Nil(t, ar)
			},
		},
		{
			name: "finds most recent analysis run",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "older-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "test-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase: "Successful",
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "newer-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "test-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase: "Failed",
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "newer-analysis", ar.Name)
				assert.Equal(t, hourAgo.Unix(), ar.CreationTimestamp.Unix())
			},
		},
		{
			name: "filters by correct stage",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "other-stage-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "other-stage",
							kargoapi.FreightCollectionLabelKey: "test-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-stage-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "correct-stage-analysis", ar.Name)
			},
		},
		{
			name: "filters by correct freight collection",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "other-freight-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "other-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-freight-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "correct-freight-analysis", ar.Name)
			},
		},
		{
			name: "handles multiple namespaces correctly",
			stage: types.NamespacedName{
				Namespace: "test-namespace",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "wrong-namespace-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "test-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-namespace-analysis",
						Namespace:         "test-namespace",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "test-namespace", ar.Namespace)
				assert.Equal(t, "correct-namespace-analysis", ar.Name)
			},
		},
		{
			name: "empty freight collection ID",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.StageLabelKey:             "test-stage",
							kargoapi.FreightCollectionLabelKey: "",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "test-analysis", ar.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&rolloutsapi.AnalysisRun{})

			if tt.interceptor.List != nil {
				builder = builder.WithInterceptorFuncs(tt.interceptor)
			}

			c := builder.Build()

			r := &RegularStagesReconciler{
				client: c,
			}

			ar, err := r.findExistingAnalysisRun(context.Background(), tt.stage, tt.freightColID)
			tt.assertions(t, ar, err)
		})
	}
}

func Test_regularStagesReconciler_autoPromoteFreight(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *fakeevent.EventRecorder, client.Client, kargoapi.StageStatus, error)
	}{
		{
			name: "no requested freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: nil,
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify no promotions were created
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name: "auto-promotion not allowed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: false,
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify no promotions were created
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name: "project not found",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "error getting Project")
			},
		},
		{
			name: "handles direct freight from warehouse",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-2",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify promotion was created for newest freight
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name: "skips promotion when current freight is latest",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"test-warehouse": {Name: "test-freight-1"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify no promotions were created
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name: "skips promotion if one already exists",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "existing-promotion",
						Labels: map[string]string{
							kargoapi.StageLabelKey: "test-stage",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-1",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify no new promotions were created
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
				assert.Equal(t, "existing-promotion", promoList.Items[0].Name)
			},
		},
		{
			name: "handles verified freight from upstream stages",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Stages: []string{"upstream-stage"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify promotion was created
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name: "handles freight approved for stage",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify promotion was created
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name: "handles multiple freight requests",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-2",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "freight-2",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-2",
					},
					Status: kargoapi.FreightStatus{},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify promotions were created for both freight items
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 2)

				// Verify they're for different freight
				freightNames := map[string]bool{}
				for _, promo := range promoList.Items {
					freightNames[promo.Spec.Freight] = true
				}
				assert.Len(t, freightNames, 2)
				assert.True(t, freightNames["freight-1"])
				assert.True(t, freightNames["freight-2"])
			},
		},
		{
			name: "creates promotion with events",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{},
				},
			},
			assertions: func(
				t *testing.T,
				e *fakeevent.EventRecorder,
				_ client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, e.Events, 1)

				event := <-e.Events
				assert.Equal(t, corev1.EventTypeNormal, event.EventType)
				assert.Equal(t, "PromotionCreated", event.Reason)
				assert.Contains(t, event.Message, "Automatically promoted Freight")
				assert.NotEmpty(t, event.Annotations)
			},
		},
		{
			name: "deduplicates freight from multiple sources",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Stages: []string{"upstream-stage"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight",
						CreationTimestamp: metav1.Time{Time: now},
					},

					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify only one promotion was created despite multiple sources
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(context.Background(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
			},
		},
		{
			name: "handles promotion creation error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake-project",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{},
				},
			},
			interceptor: interceptor.Funcs{
				Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "error creating Promotion")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				WithIndex(
					&kargoapi.Promotion{},
					indexer.PromotionsByStageAndFreightField,
					indexer.PromotionsByStageAndFreight,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByWarehouseField,
					indexer.FreightByWarehouse,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByVerifiedStagesField,
					indexer.FreightByVerifiedStages,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightApprovedForStagesField,
					indexer.FreightApprovedForStages,
				)

			c := builder.Build()
			recorder := fakeevent.NewEventRecorder(5)

			r := &RegularStagesReconciler{
				client:        c,
				eventRecorder: recorder,
			}

			status, err := r.autoPromoteFreight(context.Background(), tt.stage)
			tt.assertions(t, recorder, c, status, err)
		})
	}
}

func Test_regularStagesReconciler_autoPromotionAllowed(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stage       types.NamespacedName
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, bool, error)
	}{
		{
			name: "project not found",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.ErrorContains(t, err, "error getting Project")
				assert.False(t, allowed)
			},
		},
		{
			name: "error getting project",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.False(t, allowed)
			},
		},
		{
			name: "nil project spec",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				assert.False(t, allowed)
			},
		},
		{
			name: "empty promotion policies",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{},
					},
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				assert.False(t, allowed)
			},
		},
		{
			name: "stage not found in policies",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "other-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				assert.False(t, allowed)
			},
		},
		{
			name: "auto-promotion enabled",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				assert.True(t, allowed)
			},
		},
		{
			name: "auto-promotion disabled",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: false,
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				assert.False(t, allowed)
			},
		},
		{
			name: "multiple policies - finds correct stage",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "stage-1",
								AutoPromotionEnabled: false,
							},
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
							{
								Stage:                "stage-2",
								AutoPromotionEnabled: false,
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				assert.True(t, allowed)
			},
		},
		{
			name: "different namespace",
			stage: types.NamespacedName{
				Namespace: "other-namespace",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "other-namespace",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				assert.True(t, allowed)
			},
		},
		{
			name: "matches first policy for stage",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
					Spec: &kargoapi.ProjectSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: false,
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				assert.True(t, allowed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithInterceptorFuncs(tt.interceptor)

			c := builder.Build()

			r := &RegularStagesReconciler{
				client: c,
			}

			allowed, err := r.autoPromotionAllowed(context.Background(), tt.stage)
			tt.assertions(t, allowed, err)
		})
	}
}

func Test_summarizeConditions(t *testing.T) {
	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		status     *kargoapi.StageStatus
		err        error
		assertions func(*testing.T, *kargoapi.StageStatus)
	}{
		{
			name: "with error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{},
			err:    errors.New("something went wrong"),
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "ReconcileError", readyCond.Reason)
				assert.Equal(t, "something went wrong", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				reconcileCond := conditions.Get(status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcileCond)
				assert.Equal(t, metav1.ConditionTrue, reconcileCond.Status)
				assert.Equal(t, "RetryAfterError", reconcileCond.Reason)
				assert.Equal(t, int64(1), reconcileCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhaseFailed, status.Phase)
				assert.Equal(t, "something went wrong", status.Message)
			},
		},
		{
			name: "promoting",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypePromoting,
						Status:  metav1.ConditionTrue,
						Reason:  "Promoting",
						Message: "Stage is promoting",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "Promoting", readyCond.Reason)
				assert.Equal(t, "Stage is promoting", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhasePromoting, status.Phase)
			},
		},
		{
			name: "last promotion failed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				LastPromotion: &kargoapi.PromotionReference{
					Status: &kargoapi.PromotionStatus{
						Phase:   kargoapi.PromotionPhaseFailed,
						Message: "Promotion failed due to error",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "LastPromotionFailed", readyCond.Reason)
				assert.Equal(t, "Promotion failed due to error", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhaseFailed, status.Phase)
			},
		},
		{
			name: "unhealthy",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionFalse,
						Reason:  "HealthCheckFailed",
						Message: "Health check failed",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)

				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "HealthCheckFailed", readyCond.Reason)
				assert.Equal(t, "Health check failed", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhaseFailed, status.Phase)
			},
		},
		{
			name: "missing health condition",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "Unhealthy", readyCond.Reason)
				assert.Equal(t, "Stage is not healthy", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhaseVerifying, status.Phase)
			},
		},
		{
			name: "health unknown",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionUnknown,
						Reason:  "HealthCheckPending",
						Message: "Health check in progress",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "HealthCheckPending", readyCond.Reason)
				assert.Equal(t, "Health check in progress", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhaseVerifying, status.Phase)
			},
		},
		{
			name: "pending verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionUnknown,
						Reason:  "VerificationPending",
						Message: "Verification is pending",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "VerificationPending", readyCond.Reason)
				assert.Equal(t, "Verification is pending", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhaseVerifying, status.Phase)
			},
		},
		{
			name: "verification error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionFalse,
						Reason:  "VerificationError",
						Message: "Verification failed",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "VerificationError", readyCond.Reason)
				assert.Equal(t, "Verification failed", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhaseFailed, status.Phase)
			},
		},
		{
			name: "missing verification condition",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "PendingVerification", readyCond.Reason)
				assert.Equal(t, "Stage is not verified", readyCond.Message)
				assert.Equal(t, kargoapi.StagePhaseVerifying, status.Phase)
			},
		},
		{
			name: "ready",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionTrue,
						Reason:  "Verified",
						Message: "Stage is verified",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)

				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
				assert.Equal(t, "Verified", readyCond.Reason)
				assert.Equal(t, "Stage is verified", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, kargoapi.StagePhaseSteady, status.Phase)
				assert.Equal(t, int64(1), status.ObservedGeneration)
			},
		},
		{
			name: "reconciling condition cleared when ready",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionTrue,
						Reason:  "Verified",
						Message: "Stage is verified",
					},
					{
						Type:    kargoapi.ConditionTypeReconciling,
						Status:  metav1.ConditionTrue,
						Reason:  "Reconciling",
						Message: "Stage is reconciling",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)

				reconcileCond := conditions.Get(status, kargoapi.ConditionTypeReconciling)
				assert.Nil(t, reconcileCond, "Reconciling condition should be deleted when ready")

				assert.Equal(t, kargoapi.StagePhaseSteady, status.Phase)
				assert.Equal(t, int64(1), status.ObservedGeneration)
			},
		},
		{
			name: "freight summary updated and message cleared",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{}, {}},
				},
			},
			status: &kargoapi.StageStatus{
				Message: "Previous error message",
				FreightHistory: kargoapi.FreightHistory{
					&kargoapi.FreightCollection{
						Freight: map[string]kargoapi.FreightReference{
							"freight1": {Name: "freight1"},
						},
					},
				},
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionTrue,
						Reason:  "Verified",
						Message: "Stage is verified",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				assert.Empty(t, status.Message, "Message should be cleared")
				assert.Equal(t, "1/2 Fulfilled", status.FreightSummary)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summarizeConditions(tt.stage, tt.status, tt.err)
			tt.assertions(t, tt.status)
		})
	}
}

func Test_buildFreightSummary(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		current   *kargoapi.FreightCollection
		expected  string
	}{
		{
			name:      "nil current",
			requested: 2,
			current:   nil,
			expected:  "0/2 Fulfilled",
		},
		{
			name:      "single freight",
			requested: 1,
			current: &kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"test": {Name: "test-freight"},
				},
			},
			expected: "test-freight",
		},
		{
			name:      "multiple freight",
			requested: 3,
			current: &kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"test1": {Name: "test-freight-1"},
					"test2": {Name: "test-freight-2"},
				},
			},
			expected: "2/3 Fulfilled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFreightSummary(tt.requested, tt.current)
			assert.Equal(t, tt.expected, result)
		})
	}
}
