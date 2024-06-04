package stages

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
)

var (
	fakeTime = time.Date(2024, time.April, 10, 0, 0, 0, 0, time.UTC)
)

func TestNewReconciler(t *testing.T) {
	testCfg := ReconcilerConfig{
		RolloutsControllerInstanceID: "fake-instance-id",
	}
	kubeClient := fake.NewClientBuilder().Build()
	requirement, err := controller.GetShardRequirement(testCfg.ShardName)
	require.NoError(t, err)
	recorder := &fakeevent.EventRecorder{Events: nil}
	r := newReconciler(
		kubeClient,
		kubeClient,
		recorder,
		testCfg,
		requirement,
	)
	require.Equal(t, testCfg, r.cfg)
	require.NotNil(t, r.kargoClient)
	require.NotNil(t, r.argocdClient)
	require.NotNil(t, r.recorder)
	require.NotNil(t, r.appHealth)
	// Assert that all overridable behaviors were initialized to a default:
	// Loop guard:
	require.NotNil(t, r.nowFn)
	require.NotNil(t, r.getPromotionsForStageFn)
	require.NotNil(t, r.listPromosFn)
	require.NotNil(t, r.syncPromotionsFn)
	// Freight verification:
	require.NotNil(t, r.startVerificationFn)
	require.NotNil(t, r.getVerificationInfoFn)
	require.NotNil(t, r.getAnalysisTemplateFn)
	require.NotNil(t, r.listAnalysisRunsFn)
	require.NotNil(t, r.buildAnalysisRunFn)
	require.NotNil(t, r.createAnalysisRunFn)
	require.NotNil(t, r.getAnalysisRunFn)
	require.NotNil(t, r.getFreightFn)
	require.NotNil(t, r.verifyFreightInStageFn)
	require.NotNil(t, r.patchFreightStatusFn)
	// Auto-promotion:
	require.NotNil(t, r.isAutoPromotionPermittedFn)
	require.NotNil(t, r.getProjectFn)
	require.NotNil(t, r.createPromotionFn)
	// Discovering latest Freight:
	require.NotNil(t, r.getLatestAvailableFreightFn)
	require.NotNil(t, r.getLatestFreightFromWarehouseFn)
	require.NotNil(t, r.getAllVerifiedFreightFn)
	require.NotNil(t, r.getLatestVerifiedFreightFn)
	require.NotNil(t, r.getLatestApprovedFreightFn)
	require.NotNil(t, r.listFreightFn)
	// Stage deletion:
	require.NotNil(t, r.clearVerificationsFn)
	require.NotNil(t, r.clearApprovalsFn)
	require.NotNil(t, r.clearAnalysisRunsFn)
}

func TestSyncControlFlowStage(t *testing.T) {
	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(
			t *testing.T,
			recorder *fakeevent.EventRecorder,
			initialStatus kargoapi.StageStatus,
			newStatus kargoapi.StageStatus,
			err error,
		)
	}{
		{
			name: "error listing Freight from Warehouse",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseNotApplicable,
				},
			},
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "error listing Freight from Warehouse")
				require.ErrorContains(t, err, "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should be recorded
				require.Empty(t, recorder.Events)
			},
		},
		{
			name: "error listing Freight verified in upstream Stages",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						UpstreamStages: []kargoapi.StageSubscription{
							{Name: "fake-stage"},
						},
					},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseNotApplicable,
				},
			},
			reconciler: &reconciler{
				getAllVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(
					t, err, "error getting all Freight verified in Stages upstream from Stage",
				)
				require.ErrorContains(t, err, "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should be recorded
				require.Empty(t, recorder.Events)
			},
		},
		{
			name: "error marking Freight as verified in Stage",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseNotApplicable,
				},
			},
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{}}
					return nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "error marking Freight")
				require.ErrorContains(t, err, "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should be recorded
				require.Empty(t, recorder.Events)
			},
		},
		{
			name: "success",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 42,
				},
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseNotApplicable,
					CurrentFreight: &kargoapi.FreightReference{},
					Health:         &kargoapi.Health{},
				},
			},
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{}}
					return nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				_ kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, int64(42), newStatus.ObservedGeneration) // Set
				require.Nil(t, newStatus.CurrentFreight)                  // Cleared
				require.Nil(t, newStatus.Health)                          // Cleared

				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationSucceeded, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(1)
			testCase.reconciler.nowFn = fakeNow
			testCase.reconciler.recorder = recorder
			newStatus, err := testCase.reconciler.syncControlFlowStage(
				context.Background(),
				testCase.stage,
			)
			testCase.assertions(t, recorder, testCase.stage.Status, newStatus, err)
		})
	}
}

func TestSyncNormalStage(t *testing.T) {
	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(
			t *testing.T,
			recorder *fakeevent.EventRecorder,
			initialStatus kargoapi.StageStatus,
			newStatus kargoapi.StageStatus,
			err error,
		)
	}{
		{
			name:  "error syncing Promotions",
			stage: &kargoapi.Stage{},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should be recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "reverification requested",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: "fake-id",
					},
				},
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							ID:    "fake-id",
							Phase: kargoapi.VerificationPhaseFailed,
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name: "fake-analysis-run",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				startVerificationFn: func(
					context.Context,
					*kargoapi.Stage,
				) (*kargoapi.VerificationInfo, error) {
					return &kargoapi.VerificationInfo{
						ID:      "new-fake-id",
						Phase:   kargoapi.VerificationPhasePending,
						Message: "Awaiting reconfirmation",
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name: "new-fake-analysis-run",
						},
					}, nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				_ kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, newStatus.CurrentFreight)
				require.Equal(t, kargoapi.StagePhaseVerifying, newStatus.Phase)
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						ID:      "new-fake-id",
						Phase:   kargoapi.VerificationPhasePending,
						Message: "Awaiting reconfirmation",
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name: "new-fake-analysis-run",
						},
					},
					newStatus.CurrentFreight.VerificationInfo,
				)

				// No events should be recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "ignores reverification request if conditions are not met",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: "wrong-fake-analysis-run",
					},
				},
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							Phase: kargoapi.VerificationPhaseFailed,
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name: "fake-analysis-run",
							},
						},
						VerificationHistory: []kargoapi.VerificationInfo{
							{
								Phase: kargoapi.VerificationPhaseFailed,
								AnalysisRun: &kargoapi.AnalysisRunReference{
									Name: "fake-analysis-run",
								},
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, newStatus.CurrentFreight)
				require.Equal(t, initialStatus, newStatus)

				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "error starting verification",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseVerifying,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				startVerificationFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
				) (*kargoapi.VerificationInfo, error) {
					return &kargoapi.VerificationInfo{
						Phase:      kargoapi.VerificationPhaseError,
						Message:    "something went wrong",
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, newStatus.CurrentFreight)
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)

				expectInfo := kargoapi.VerificationInfo{
					StartTime:  ptr.To(metav1.NewTime(fakeTime)),
					FinishTime: ptr.To(metav1.NewTime(fakeTime)),
					Phase:      kargoapi.VerificationPhaseError,
					Message:    "something went wrong",
				}

				require.Equal(
					t,
					&expectInfo,
					newStatus.CurrentFreight.VerificationInfo,
				)
				require.Equal(
					t,
					kargoapi.VerificationInfoStack{expectInfo},
					newStatus.CurrentFreight.VerificationHistory,
				)
				// Everything else should be returned unchanged
				newStatus.CurrentFreight.VerificationInfo = nil
				newStatus.CurrentFreight.VerificationHistory = nil
				newStatus.Phase = initialStatus.Phase
				require.Equal(t, initialStatus, newStatus)

				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationErrored, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)
			},
		},

		{
			name: "retryable error starting verification",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseVerifying,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				startVerificationFn: func(
					context.Context,
					*kargoapi.Stage,
				) (*kargoapi.VerificationInfo, error) {
					return &kargoapi.VerificationInfo{
						Phase:   kargoapi.VerificationPhaseError,
						Message: "something went wrong",
					}, errors.New("retryable error")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "retryable error")
				require.NotNil(t, newStatus.CurrentFreight)
				require.Equal(t, kargoapi.StagePhaseVerifying, newStatus.Phase)

				expectInfo := kargoapi.VerificationInfo{
					Phase:   kargoapi.VerificationPhaseError,
					Message: "something went wrong",
				}

				require.Equal(
					t,
					&expectInfo,
					newStatus.CurrentFreight.VerificationInfo,
				)
				require.Equal(
					t,
					kargoapi.VerificationInfoStack{expectInfo},
					newStatus.CurrentFreight.VerificationHistory,
				)

				// Everything else should be returned unchanged
				newStatus.CurrentFreight.VerificationInfo = nil
				newStatus.CurrentFreight.VerificationHistory = nil
				newStatus.Phase = initialStatus.Phase
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error checking verification result",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseVerifying,
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							Phase: kargoapi.VerificationPhasePending,
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-analysis-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				getVerificationInfoFn: func(_ context.Context, _ *kargoapi.Stage) (*kargoapi.VerificationInfo, error) {
					return &kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseError,
						Message:    "something went wrong",
					}, nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, newStatus.CurrentFreight)
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseError,
						Message:    "something went wrong",
					},
					newStatus.CurrentFreight.VerificationInfo,
				)
				// Phase should be changed to Steady
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)
				// Everything else should be unchanged
				newStatus.Phase = initialStatus.Phase
				newStatus.CurrentFreight = initialStatus.CurrentFreight
				require.Equal(t, initialStatus, newStatus)

				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationErrored, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)
			},
		},

		{
			name: "retryable error checking verification result",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseVerifying,
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							Phase: kargoapi.VerificationPhasePending,
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-analysis-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				getVerificationInfoFn: func(_ context.Context, _ *kargoapi.Stage) (*kargoapi.VerificationInfo, error) {
					return &kargoapi.VerificationInfo{
						Phase:   kargoapi.VerificationPhaseError,
						Message: "something went wrong",
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "fake-analysis-run",
							Namespace: "fake-namespace",
						},
					}, errors.New("retryable error")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "retryable error")
				require.NotNil(t, newStatus.CurrentFreight)
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						Phase:   kargoapi.VerificationPhaseError,
						Message: "something went wrong",
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "fake-analysis-run",
							Namespace: "fake-namespace",
						},
					},
					newStatus.CurrentFreight.VerificationInfo,
				)
				require.Len(t, newStatus.CurrentFreight.VerificationHistory, 1)

				// Phase should not be changed to Steady
				require.Equal(t, kargoapi.StagePhaseVerifying, newStatus.Phase)
				// Everything else should be unchanged
				newStatus.Phase = initialStatus.Phase
				newStatus.CurrentFreight = initialStatus.CurrentFreight
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "verification aborted",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "fake-id",
					},
				},
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseVerifying,
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							ID:    "fake-id",
							Phase: kargoapi.VerificationPhasePending,
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name: "fake-analysis-run",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				getVerificationInfoFn: func(
					_ context.Context,
					s *kargoapi.Stage,
				) (*kargoapi.VerificationInfo, error) {
					return s.Status.CurrentFreight.VerificationInfo, nil
				},
				abortVerificationFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
				) *kargoapi.VerificationInfo {
					return &kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseAborted,
						Message:    "aborted",
					}
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				_ kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, newStatus.CurrentFreight)
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseAborted,
						Message:    "aborted",
					},
					newStatus.CurrentFreight.VerificationInfo,
				)

				// Phase should be changed to Steady
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)

				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationAborted, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)
			},
		},

		{
			name: "verification abort conditions not met",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "fake-id",
					},
				},
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseVerifying,
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							ID:        "fake-id",
							StartTime: ptr.To(metav1.NewTime(fakeTime)),
							Phase:     kargoapi.VerificationPhasePending,
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name: "fake-analysis-run",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-analysis-run",
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				getVerificationInfoFn: func(
					_ context.Context,
					s *kargoapi.Stage,
				) (*kargoapi.VerificationInfo, error) {
					i := s.Status.CurrentFreight.VerificationInfo.DeepCopy()
					i.FinishTime = ptr.To(metav1.NewTime(fakeTime))
					i.Phase = kargoapi.VerificationPhaseError
					return i, nil
				},
				abortVerificationFn: func(
					context.Context,
					*kargoapi.Stage,
				) *kargoapi.VerificationInfo {
					// Should not be called
					return &kargoapi.VerificationInfo{
						Phase:      kargoapi.VerificationPhaseAborted,
						FinishTime: ptr.To(metav1.NewTime(time.Now())),
						Message:    "aborted",
					}
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				_ kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, newStatus.CurrentFreight)
				require.Equal(t, kargoapi.VerificationPhaseError, newStatus.CurrentFreight.VerificationInfo.Phase)

				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationErrored, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)
			},
		},

		{
			name: "error marking Freight as verified in Stage",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseVerifying,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return false, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error marking Freight")
				// Since no verification process was defined and the Stage is healthy,
				// the Stage should have transitioned to a Steady phase.
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)
				// Status should be otherwise unchanged
				newStatus.Phase = initialStatus.Phase
				require.Equal(t, initialStatus, newStatus)

				// No events should be recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "error checking if auto-promotion is permitted",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return true, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return false, errors.New("something went wrong")
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				// Verification should be done before auto-promotion
				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationSucceeded, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)

				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error checking if auto-promotion is permitted")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "auto-promotion is not permitted",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return true, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return false, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				// Verification should be done before auto-promotion
				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationSucceeded, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)

				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error getting latest Freight",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return true, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					*kargoapi.Stage,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				// Verification should be done before auto-promotion
				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationSucceeded, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)

				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error finding latest Freight for Stage")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "no Freight found",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return true, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					*kargoapi.Stage,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should be recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "Stage already has latest Freight",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{
						Name: "fake-freight-id",
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return true, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					*kargoapi.Stage,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "Promotion already exists",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return true, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					*kargoapi.Stage,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				listPromosFn: func(
					_ context.Context,
					obj client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := obj.(*kargoapi.PromotionList)
					require.True(t, ok)
					promos.Items = []kargoapi.Promotion{{}}
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error creating Promotion",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase:          kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return true, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					*kargoapi.Stage,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				listPromosFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				// Verification should be done before promotion
				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationSucceeded, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)

				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error creating Promotion of Stage")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "skip event recording if no verification performed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 42,
					Name:       "fake-stage",
				},
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							Phase: kargoapi.VerificationPhaseSuccessful,
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-analysis-run",
								Namespace: "fake-namespace",
								Phase:     string(rollouts.AnalysisPhaseSuccessful),
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
				},
				getVerificationInfoFn: func(
					context.Context,
					*kargoapi.Stage,
				) (*kargoapi.VerificationInfo, error) {
					return &kargoapi.VerificationInfo{
						Phase: kargoapi.VerificationPhaseSuccessful,
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "fake-analysis-run",
							Namespace: "fake-namespace",
							Phase:     string(rollouts.AnalysisPhaseSuccessful),
						},
					}, nil
				},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					// No updates are performed
					return false, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return false, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					*kargoapi.Stage,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				listPromosFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				_ kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, int64(42), newStatus.ObservedGeneration) // Set
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)
				require.NotNil(t, newStatus.Health) // Set
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						Phase: kargoapi.VerificationPhaseSuccessful,
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "fake-analysis-run",
							Namespace: "fake-namespace",
							Phase:     string(rollouts.AnalysisPhaseSuccessful),
						},
					},
					newStatus.CurrentFreight.VerificationInfo,
				)
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "success",
			// Note: In this final case, we will also assert than anything that should
			// have been cleared or modified in the Stage's status was.
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 42,
				},
				Spec: kargoapi.StageSpec{
					Subscriptions: kargoapi.Subscriptions{
						Warehouse: "fake-warehouse",
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseVerifying,
					CurrentFreight: &kargoapi.FreightReference{
						VerificationInfo: &kargoapi.VerificationInfo{
							Phase: kargoapi.VerificationPhasePending,
							AnalysisRun: &kargoapi.AnalysisRunReference{
								Name:      "fake-analysis-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				syncPromotionsFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					status kargoapi.StageStatus,
				) (kargoapi.StageStatus, error) {
					return status, nil
				},
				appHealth: &mockAppHealthEvaluator{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
				},
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-namespace",
							Name:      "fake-analysis-run",
						},
					}, nil
				},
				getVerificationInfoFn: func(
					context.Context,
					*kargoapi.Stage,
				) (*kargoapi.VerificationInfo, error) {
					return &kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseSuccessful,
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "fake-analysis-run",
							Namespace: "fake-namespace",
							Phase:     string(rollouts.AnalysisPhaseSuccessful),
						},
					}, nil
				},
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return true, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				getLatestAvailableFreightFn: func(
					context.Context,
					string,
					*kargoapi.Stage,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}, nil
				},
				listPromosFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				_ kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, int64(42), newStatus.ObservedGeneration) // Set
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)
				require.NotNil(t, newStatus.Health) // Set
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseSuccessful,
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "fake-analysis-run",
							Namespace: "fake-namespace",
							Phase:     string(rollouts.AnalysisPhaseSuccessful),
						},
					},
					newStatus.CurrentFreight.VerificationInfo,
				)

				require.Len(t, recorder.Events, 2)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonFreightVerificationSucceeded, event.Reason)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				require.Equal(t,
					fakeTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)

				// The second event should be the promotion creation event (auto-promotion)
				event = <-recorder.Events
				require.Equal(t, kargoapi.EventReasonPromotionCreated, event.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(2)
			testCase.reconciler.nowFn = fakeNow
			testCase.reconciler.recorder = recorder
			newStatus, err := testCase.reconciler.syncNormalStage(
				context.Background(),
				testCase.stage,
			)
			testCase.assertions(t, recorder, testCase.stage.Status, newStatus, err)
		})
	}
}

func TestReconciler_syncPromotions(t *testing.T) {
	now := fakeNow()
	ulidOneMinuteAgo := ulid.MustNew(ulid.Timestamp(now.Add(-time.Minute)), nil)
	ulidOneHourAgo := ulid.MustNew(ulid.Timestamp(now.Add(-time.Hour)), nil)
	ulidOneDayAgo := ulid.MustNew(ulid.Timestamp(now.Add(-24*time.Hour)), nil)

	testCases := []struct {
		name          string
		reconciler    *reconciler
		initialStatus kargoapi.StageStatus
		assertions    func(*testing.T, kargoapi.StageStatus, error)
	}{
		{
			name: "error listing Promotions",
			reconciler: &reconciler{
				getPromotionsForStageFn: func(context.Context, string, string) ([]kargoapi.Promotion, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ kargoapi.StageStatus, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "no Promotions found",
			reconciler: &reconciler{
				getPromotionsForStageFn: func(context.Context, string, string) ([]kargoapi.Promotion, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "latest Promotion is Running",
			reconciler: &reconciler{
				getPromotionsForStageFn: func(context.Context, string, string) ([]kargoapi.Promotion, error) {
					return []kargoapi.Promotion{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "fake-promotion",
								CreationTimestamp: metav1.NewTime(now),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseRunning,
								Freight: &kargoapi.FreightReference{
									Name: "fake-freight",
								},
							},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.StagePhasePromoting, status.Phase)
				require.Equal(t, &kargoapi.PromotionInfo{
					Name: "fake-promotion",
					Freight: kargoapi.FreightReference{
						Name: "fake-freight",
					},
				}, status.CurrentPromotion)
			},
		},
		{
			name: "latest Promotion is not Running anymore",
			reconciler: &reconciler{
				getPromotionsForStageFn: func(context.Context, string, string) ([]kargoapi.Promotion, error) {
					return []kargoapi.Promotion{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "fake-promotion",
								CreationTimestamp: metav1.NewTime(now),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseSucceeded,
								Freight: &kargoapi.FreightReference{
									Name: "fake-freight",
								},
							},
						},
					}, nil
				},
			},
			initialStatus: kargoapi.StageStatus{
				Phase: kargoapi.StagePhasePromoting,
				CurrentPromotion: &kargoapi.PromotionInfo{
					Name: "fake-promotion",
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.StagePhaseSteady, status.Phase)
				require.Nil(t, status.CurrentPromotion)
			},
		},
		{
			name: "new Terminated Promotions",
			reconciler: &reconciler{
				getPromotionsForStageFn: func(context.Context, string, string) ([]kargoapi.Promotion, error) {
					return []kargoapi.Promotion{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-promotion." + ulidOneDayAgo.String(),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseSucceeded,
								Freight: &kargoapi.FreightReference{
									Name: "fake-freight-1",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-promotion." + ulidOneMinuteAgo.String(),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseErrored,
								Freight: &kargoapi.FreightReference{
									Name: "fake-freight-3",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-promotion." + ulidOneHourAgo.String(),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseFailed,
								Freight: &kargoapi.FreightReference{
									Name: "fake-freight-2",
								},
							},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)

				require.Equal(t, kargoapi.StagePhaseSteady, status.Phase)
				require.Nil(t, status.CurrentPromotion)

				// Last Promotion should be the latest Terminated Promotion
				require.Equal(t, &kargoapi.PromotionInfo{
					Name: "fake-promotion." + ulidOneMinuteAgo.String(),
					Status: &kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseErrored,
						Freight: &kargoapi.FreightReference{
							Name: "fake-freight-3",
						},
					},
					Freight: kargoapi.FreightReference{
						Name: "fake-freight-3",
					},
				}, status.LastPromotion)

				// Current Freight should be the Freight of the last Succeeded Promotion
				require.Equal(t, &kargoapi.FreightReference{
					Name: "fake-freight-1",
				}, status.CurrentFreight)
				require.Equal(t, kargoapi.FreightReferenceStack{
					{Name: "fake-freight-1"},
				}, status.History)
			},
		},
		{
			name: "no new Terminated Promotions",
			reconciler: &reconciler{
				getPromotionsForStageFn: func(context.Context, string, string) ([]kargoapi.Promotion, error) {
					return []kargoapi.Promotion{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-promotion." + ulidOneDayAgo.String(),
							},
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseSucceeded,
								Freight: &kargoapi.FreightReference{
									Name: "fake-freight-1",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-promotion." + ulidOneHourAgo.String(),
							},
							Status: kargoapi.PromotionStatus{
								// Phase update should be ignored
								Phase: kargoapi.PromotionPhaseSucceeded,
								Freight: &kargoapi.FreightReference{
									Name: "fake-freight-2",
								},
							},
						},
					}, nil
				},
			},
			initialStatus: kargoapi.StageStatus{
				// Should not be updated.
				Phase: kargoapi.StagePhaseVerifying,
				LastPromotion: &kargoapi.PromotionInfo{
					Name: "fake-promotion." + ulidOneHourAgo.String(),
					Status: &kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseFailed,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)

				require.Equal(t, kargoapi.StagePhaseVerifying, status.Phase)
				require.Nil(t, status.CurrentPromotion)
				require.Equal(t, &kargoapi.PromotionInfo{
					Name: "fake-promotion." + ulidOneHourAgo.String(),
					Status: &kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseFailed,
					},
				}, status.LastPromotion)
				require.Len(t, status.History, 0)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := testCase.reconciler.syncPromotions(
				context.Background(),
				&kargoapi.Stage{},
				testCase.initialStatus,
			)
			testCase.assertions(t, status, err)
		})
	}
}

func TestSyncStageDelete(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(
			t *testing.T,
			initialStatus kargoapi.StageStatus,
			newStatus kargoapi.StageStatus,
			err error,
		)
	}{
		{
			name: "error clearing verifications",
			reconciler: &reconciler{
				clearVerificationsFn: func(context.Context, *kargoapi.Stage) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, initialStatus, newStatus kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "error clearing verifications for Stage")
				require.ErrorContains(t, err, "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "error clearing approvals",
			reconciler: &reconciler{
				clearVerificationsFn: func(context.Context, *kargoapi.Stage) error {
					return nil
				},
				clearApprovalsFn: func(context.Context, *kargoapi.Stage) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, initialStatus, newStatus kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "error clearing approvals for Stage")
				require.ErrorContains(t, err, "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "error clearing AnalysisRuns",
			reconciler: &reconciler{
				clearVerificationsFn: func(context.Context, *kargoapi.Stage) error { return nil },
				clearApprovalsFn:     func(context.Context, *kargoapi.Stage) error { return nil },
				clearAnalysisRunsFn: func(context.Context, *kargoapi.Stage) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, initialStatus, newStatus kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "error clearing AnalysisRuns for Stage")
				require.ErrorContains(t, err, "something went wrong")
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				clearVerificationsFn: func(context.Context, *kargoapi.Stage) error {
					return nil
				},
				clearApprovalsFn: func(context.Context, *kargoapi.Stage) error {
					return nil
				},
				clearAnalysisRunsFn: func(context.Context, *kargoapi.Stage) error { return nil },
			},
			assertions: func(t *testing.T, initialStatus, newStatus kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testStage := &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{kargoapi.FinalizerName},
				},
			}
			newStatus, err := testCase.reconciler.syncStageDelete(context.Background(), testStage)
			testCase.assertions(t, testStage.Status, newStatus, err)
		})
	}
}

func TestClearVerification(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, error)
	}{
		{
			name: "error listing verified Freight",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error listing Freight verified in Stage")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error patching Freight status",
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{},
						},
					}}
					return nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error patching status of Freight")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{},
						},
					}}
					return nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
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
				testCase.reconciler.clearVerifications(
					context.Background(),
					&kargoapi.Stage{},
				),
			)
		})
	}
}

func TestClearApprovals(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, error)
	}{
		{
			name: "error listing approved Freight",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error listing Freight approved for Stage")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error patching Freight status",
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{
						Status: kargoapi.FreightStatus{
							ApprovedFor: map[string]kargoapi.ApprovedStage{},
						},
					}}
					return nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error patching status of Freight")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{
						Status: kargoapi.FreightStatus{
							ApprovedFor: map[string]kargoapi.ApprovedStage{},
						},
					}}
					return nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
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
				testCase.reconciler.clearApprovals(
					context.Background(),
					&kargoapi.Stage{},
				),
			)
		})
	}
}

func TestVerifyFreightInStage(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, bool, error)
	}{
		{
			name: "error getting Freight",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, updated bool, err error) {
				require.False(t, updated)
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error finding Freight")
			},
		},
		{
			name: "Freight not found",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, updated bool, err error) {
				require.False(t, updated)
				require.ErrorContains(t, err, "found no Freight")
			},
		},
		{
			name: "Freight already verified in Stage",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, updated bool, err error) {
				require.NoError(t, err)
				require.False(t, updated)
			},
		},
		{
			name: "error Patching Freight status",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, updated bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.False(t, updated)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, updated bool, err error) {
				require.NoError(t, err)
				require.True(t, updated)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updated, err := testCase.reconciler.verifyFreightInStage(
				context.Background(),
				"fake-namespace",
				"fake-freight",
				"fake-stage",
			)
			testCase.assertions(t, updated, err)
		})
	}
}

func TestIsAutoPromotionPermitted(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, bool, error)
	}{
		{
			name: "error getting Project",
			reconciler: &reconciler{
				getProjectFn: func(
					context.Context,
					client.Client,
					string,
				) (*kargoapi.Project, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error finding Project")
				require.False(t, allowed)
			},
		},
		{
			name: "no Project found",
			reconciler: &reconciler{
				getProjectFn: func(
					context.Context,
					client.Client,
					string,
				) (*kargoapi.Project, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, allowed bool, err error) {
				require.ErrorContains(t, err, "Project")
				require.ErrorContains(t, err, "not found")
				require.False(t, allowed)
			},
		},
		{
			name: "defaults to not permitted",
			reconciler: &reconciler{
				getProjectFn: func(_ context.Context, _ client.Client, _ string) (*kargoapi.Project, error) {
					return &kargoapi.Project{}, nil
				},
			},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
		{
			name: "explicitly not permitted",
			reconciler: &reconciler{
				getProjectFn: func(_ context.Context, _ client.Client, _ string) (*kargoapi.Project, error) {
					return &kargoapi.Project{
						Spec: &kargoapi.ProjectSpec{
							PromotionPolicies: []kargoapi.PromotionPolicy{
								{
									Stage:                "fake-stage",
									AutoPromotionEnabled: false,
								},
							},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
		{
			name: "permitted",
			reconciler: &reconciler{
				getProjectFn: func(_ context.Context, _ client.Client, _ string) (*kargoapi.Project, error) {
					return &kargoapi.Project{
						Spec: &kargoapi.ProjectSpec{
							PromotionPolicies: []kargoapi.PromotionPolicy{
								{
									Stage:                "fake-stage",
									AutoPromotionEnabled: true,
								},
							},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.True(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.reconciler.isAutoPromotionPermitted(
				context.Background(),
				"fake-namespace",
				"fake-stage",
			)
			testCase.assertions(t, res, err)
		})
	}
}

func TestGetPromotionsForStage(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, []kargoapi.Promotion, error)
	}{
		{
			name: "error listing Promotions",
			reconciler: &reconciler{
				listPromosFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Promotion, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error listing Promotions")
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				listPromosFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, promos []kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Empty(t, promos)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			promos, err := testCase.reconciler.getPromotionsForStage(
				context.Background(),
				"fake-namespace",
				"fake-stage",
			)
			testCase.assertions(t, promos, err)
		})
	}
}

func TestGetLatestAvailableFreight(t *testing.T) {
	now := time.Now().UTC()
	testCases := []struct {
		name       string
		subs       kargoapi.Subscriptions
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name: "error getting latest Freight from Warehouse",
			subs: kargoapi.Subscriptions{
				Warehouse: "fake-warehouse",
			},
			reconciler: &reconciler{
				getLatestFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error checking Warehouse")
			},
		},
		{
			name: "found no Freight from Warehouse",
			subs: kargoapi.Subscriptions{
				Warehouse: "fake-warehouse",
			},
			reconciler: &reconciler{
				getLatestFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success getting latest Freight from Warehouse",
			subs: kargoapi.Subscriptions{
				Warehouse: "fake-warehouse",
			},
			reconciler: &reconciler{
				getLatestFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
			},
		},
		{
			name: "error getting latest Freight verified in upstream Stages",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(
					t, err, "error finding latest Freight verified in Stages upstream from Stage",
				)
			},
		},
		{
			name: "error getting latest Freight approved for Stage",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
				getLatestApprovedFreightFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error finding latest Freight approved for Stage")
			},
		},
		{
			name: "found no suitable Freight",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
				getLatestApprovedFreightFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "only found latest Freight verified in upstream Stages",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				getLatestApprovedFreightFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
			},
		},
		{
			name: "only found latest Freight approved for Stage",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
				getLatestApprovedFreightFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
			},
		},
		{
			name: "latest verified Freight is newer than latest approved Freight",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "newer-freight",
							CreationTimestamp: metav1.Time{
								Time: now,
							},
						},
					}, nil
				},
				getLatestApprovedFreightFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "older-freight",
							CreationTimestamp: metav1.Time{
								Time: now.Add(-time.Hour),
							},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				require.Equal(t, "newer-freight", freight.Name)
			},
		},
		{
			name: "latest approved Freight is newer than latest verified Freight",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{{}},
			},
			reconciler: &reconciler{
				getLatestVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "older-freight",
							CreationTimestamp: metav1.Time{
								Time: now.Add(-time.Hour),
							},
						},
					}, nil
				},
				getLatestApprovedFreightFn: func(
					context.Context,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "newer-freight",
							CreationTimestamp: metav1.Time{
								Time: now,
							},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				require.Equal(t, "newer-freight", freight.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.reconciler.getLatestAvailableFreight(
				context.Background(),
				"fake-namespace",
				&kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						Subscriptions: testCase.subs,
					},
				},
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGetLatestFreightFromWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name: "error listing Freight from Warehouse",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "no Freight found from Warehouse",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "newer-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "older-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now().Add(-time.Hour),
								},
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				// Be sure we got the latest
				require.Equal(t, "newer-freight", freight.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.reconciler.getLatestFreightFromWarehouse(
				context.Background(),
				"fake-namespace",
				"fake-warehouse",
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGetAllVerifiedFreight(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "error listing Freight",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error listing Freight verified in Stage")
			},
		},
		{
			name: "no Freight found",
			reconciler: &reconciler{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-freight",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "another-fake-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				require.Len(t, freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.reconciler.getAllVerifiedFreight(
				context.Background(),
				"fake-namespace",
				[]kargoapi.StageSubscription{
					{
						Name: "fake-stage",
					},
				},
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGetLatestVerifiedFreight(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name: "error getting all Freight verified in upstream Stages",
			reconciler: &reconciler{
				getAllVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "no Freight verified in upstream Stages",
			reconciler: &reconciler{
				getAllVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				getAllVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "newer-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "older-freight",
								CreationTimestamp: metav1.Time{
									Time: time.Now().Add(-time.Hour),
								},
							},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				// Be sure we got the latest
				require.Equal(t, "newer-freight", freight.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.reconciler.getLatestVerifiedFreight(
				context.Background(),
				"fake-namespace",
				[]kargoapi.StageSubscription{},
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func fakeNow() time.Time {
	return fakeTime
}
