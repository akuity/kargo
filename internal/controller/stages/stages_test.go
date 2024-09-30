package stages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/directives"
	"github.com/akuity/kargo/internal/kubeclient"
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
	directivesEngine := &directives.FakeEngine{}
	recorder := &fakeevent.EventRecorder{Events: nil}
	r := newReconciler(
		kubeClient,
		kubeClient,
		directivesEngine,
		recorder,
		testCfg,
		requirement,
	)
	require.Equal(t, testCfg, r.cfg)
	require.NotNil(t, r.kargoClient)
	require.NotNil(t, r.directivesEngine)
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
	// Discovering Freight:
	require.NotNil(t, r.getAvailableFreightFn)
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
			name: "error getting available Freight",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseNotApplicable,
				},
			},
			reconciler: &reconciler{
				getAvailableFreightFn: func(
					context.Context, *kargoapi.Stage, bool,
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
				require.ErrorContains(t, err, "error getting available Freight for control flow Stage")
				require.ErrorContains(t, err, "something went wrong")
				newStatus.FreightSummary = ""
				// Status should be otherwise unchanged
				require.Equal(t, initialStatus, newStatus)
				// No events should be recorded
				require.Empty(t, recorder.Events)
			},
		},
		{
			name: "error marking Freight as verified in Stage",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseNotApplicable,
				},
			},
			reconciler: &reconciler{
				getAvailableFreightFn: func(
					context.Context, *kargoapi.Stage, bool,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{{}}, nil
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
				newStatus.FreightSummary = ""
				// Status should be otherwise unchanged
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
				Status: kargoapi.StageStatus{
					Phase:            kargoapi.StagePhaseNotApplicable,
					CurrentPromotion: &kargoapi.PromotionReference{},
					LastPromotion:    &kargoapi.PromotionReference{},
					FreightHistory:   make(kargoapi.FreightHistory, 0),
					Health:           &kargoapi.Health{},
				},
			},
			reconciler: &reconciler{
				getAvailableFreightFn: func(
					context.Context, *kargoapi.Stage, bool,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{{}}, nil
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
				require.Nil(t, newStatus.FreightHistory)                  // Cleared
				require.Nil(t, newStatus.CurrentPromotion)                // Cleared
				require.Nil(t, newStatus.LastPromotion)                   // Cleared
				require.Nil(t, newStatus.Health)                          // Cleared
				require.Equal(t, "N/A", newStatus.FreightSummary)

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
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
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
				require.ErrorContains(t, err, "something went wrong")

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
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
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
								},
							},
							VerificationHistory: []kargoapi.VerificationInfo{{
								ID:    "fake-id",
								Phase: kargoapi.VerificationPhaseFailed,
								AnalysisRun: &kargoapi.AnalysisRunReference{
									Name: "fake-analysis-run",
								},
							}},
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
					*kargoapi.FreightCollection,
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
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
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
					newStatus.FreightHistory.Current().VerificationHistory.Current(),
				)

				// Status should be otherwise unchanged
				newStatus.Phase = initialStatus.Phase
				newStatus.FreightHistory.Current().VerificationHistory =
					initialStatus.FreightHistory.Current().VerificationHistory
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
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
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
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
					Phase: kargoapi.StagePhaseVerifying,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				startVerificationFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.FreightCollection,
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
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)

				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseError,
						Message:    "something went wrong",
					},
					newStatus.FreightHistory.Current().VerificationHistory.Current(),
				)

				// Status should be otherwise unchanged
				newStatus.Phase = initialStatus.Phase
				newStatus.FreightHistory.Current().VerificationHistory =
					initialStatus.FreightHistory.Current().VerificationHistory
				require.Equal(t, initialStatus, newStatus)

				// The unrecoverable error should have been recorded as an event
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
					Phase: kargoapi.StagePhaseVerifying,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				startVerificationFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.FreightCollection,
				) (*kargoapi.VerificationInfo, error) {
					return &kargoapi.VerificationInfo{
						Phase:   kargoapi.VerificationPhaseError,
						Message: "something went wrong",
					}, errors.New("retryable error")
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "retryable error")

				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						Phase:   kargoapi.VerificationPhaseError,
						Message: "something went wrong",
					},
					newStatus.FreightHistory.Current().VerificationHistory.Current(),
				)

				// Status should be otherwise unchanged
				newStatus.FreightHistory.Current().VerificationHistory =
					initialStatus.FreightHistory.Current().VerificationHistory
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
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
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
								},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhasePending,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "fake-analysis-run",
										Namespace: "fake-namespace",
									},
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
				getVerificationInfoFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.VerificationInfo,
				) (*kargoapi.VerificationInfo, error) {
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
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)

				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseError,
						Message:    "something went wrong",
					},
					newStatus.FreightHistory.Current().VerificationHistory.Current(),
				)

				// Status should be otherwise unchanged
				newStatus.Phase = initialStatus.Phase
				newStatus.FreightHistory.Current().VerificationHistory =
					initialStatus.FreightHistory.Current().VerificationHistory
				require.Equal(t, initialStatus, newStatus)

				// The unrecoverable error should have been recorded as an event
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
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
								},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhasePending,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "fake-analysis-run",
										Namespace: "fake-namespace",
									},
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
				getVerificationInfoFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.VerificationInfo,
				) (*kargoapi.VerificationInfo, error) {
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
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "retryable error")

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
					newStatus.FreightHistory.Current().VerificationHistory.Current(),
				)

				// Status should be otherwise unchanged
				newStatus.FreightHistory.Current().VerificationHistory =
					initialStatus.FreightHistory.Current().VerificationHistory
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
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
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
								},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "fake-id",
									Phase: kargoapi.VerificationPhasePending,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name: "fake-analysis-run",
									},
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
					_ *kargoapi.VerificationInfo,
				) (*kargoapi.VerificationInfo, error) {
					return s.Status.FreightHistory.Current().VerificationHistory.Current(), nil
				},
				abortVerificationFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					_ *kargoapi.VerificationInfo,
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
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)

				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(fakeTime)),
						FinishTime: ptr.To(metav1.NewTime(fakeTime)),
						Phase:      kargoapi.VerificationPhaseAborted,
						Message:    "aborted",
					},
					newStatus.FreightHistory.Current().VerificationHistory.Current(),
				)

				// Status should be otherwise unchanged
				newStatus.Phase = kargoapi.StagePhaseVerifying
				newStatus.FreightHistory.Current().VerificationHistory =
					initialStatus.FreightHistory.Current().VerificationHistory

				// The aborted verification should have been recorded as an event
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
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
								},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
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
					_ *kargoapi.VerificationInfo,
				) (*kargoapi.VerificationInfo, error) {
					i := s.Status.FreightHistory.Current().VerificationHistory.Current().DeepCopy()
					i.FinishTime = ptr.To(metav1.NewTime(fakeTime))
					i.Phase = kargoapi.VerificationPhaseError
					return i, nil
				},
				abortVerificationFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.VerificationInfo,
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
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)

				require.Equal(
					t,
					kargoapi.VerificationPhaseError,
					newStatus.FreightHistory.Current().VerificationHistory.Current().Phase,
				)

				// Status should be otherwise unchanged
				newStatus.Phase = kargoapi.StagePhaseVerifying
				newStatus.FreightHistory.Current().VerificationHistory =
					initialStatus.FreightHistory.Current().VerificationHistory
				require.Equal(t, initialStatus, newStatus)

				// The verification error should have been recorded as an event
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
					Phase: kargoapi.StagePhaseVerifying,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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

				// No events should have been recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "error checking if auto-promotion is permitted",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight:    []kargoapi.FreightRequest{{}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return false, nil
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
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error checking if auto-promotion is permitted")

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "auto-promotion is not permitted",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight:    []kargoapi.FreightRequest{{}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
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

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "error getting available Freight",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight:    []kargoapi.FreightRequest{{}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return false, nil
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
				getAvailableFreightByOriginFn: func(
					context.Context, *kargoapi.Stage, bool,
				) (map[string][]kargoapi.Freight, error) {
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
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error finding latest Freight for Stage")

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "no Freight found",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight:    []kargoapi.FreightRequest{{}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
								},
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
				getAvailableFreightByOriginFn: func(
					context.Context, *kargoapi.Stage, bool,
				) (map[string][]kargoapi.Freight, error) {
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

				// No events should have been recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "Stage already has latest Freight",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: testOrigin.Name,
							},
						},
					},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Name:   "fake-freight-id",
									Origin: testOrigin,
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
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return false, nil
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
				getAvailableFreightByOriginFn: func(
					context.Context, *kargoapi.Stage, bool,
				) (map[string][]kargoapi.Freight, error) {
					return map[string][]kargoapi.Freight{
						testOrigin.String(): {
							{
								ObjectMeta: metav1.ObjectMeta{
									Name: "fake-freight-id",
								},
							},
						},
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

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "Promotion already exists",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight:    []kargoapi.FreightRequest{{}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return false, nil
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
				getAvailableFreightByOriginFn: func(
					context.Context, *kargoapi.Stage, bool,
				) (map[string][]kargoapi.Freight, error) {
					return map[string][]kargoapi.Freight{
						testOrigin.String(): {
							{
								ObjectMeta: metav1.ObjectMeta{
									Name: "fake-freight-id",
								},
							},
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
				recorder *fakeevent.EventRecorder,
				initialStatus kargoapi.StageStatus,
				newStatus kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "error listing Promotions",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight:    []kargoapi.FreightRequest{{}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return false, nil
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
				getAvailableFreightByOriginFn: func(
					context.Context, *kargoapi.Stage, bool,
				) (map[string][]kargoapi.Freight, error) {
					return map[string][]kargoapi.Freight{
						testOrigin.String(): {
							{
								ObjectMeta: metav1.ObjectMeta{
									Name: "fake-freight-id",
								},
							},
						},
					}, nil
				},
				listPromosFn: func(
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
				require.ErrorContains(t, err, "something went wrong")

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
			},
		},

		{
			name: "error creating Promotion",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight:    []kargoapi.FreightRequest{{}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return false, nil
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
				getAvailableFreightByOriginFn: func(
					context.Context, *kargoapi.Stage, bool,
				) (map[string][]kargoapi.Freight, error) {
					return map[string][]kargoapi.Freight{
						testOrigin.String(): {
							{
								ObjectMeta: metav1.ObjectMeta{
									Name: "fake-freight-id",
								},
							},
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
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error creating Promotion of Stage")

				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)

				// No events should have been recorded
				require.Empty(t, recorder.Events)
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
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
								},
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
					*kargoapi.VerificationInfo,
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
				getAvailableFreightFn: func(
					context.Context, *kargoapi.Stage, bool,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight-id",
						},
					}}, nil
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

				// Status should be returned unchanged
				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)

				// No events should have been recorded
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
					RequestedFreight:    []kargoapi.FreightRequest{{}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					Verification:        &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseVerifying,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
								},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhasePending,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "fake-analysis-run",
										Namespace: "fake-namespace",
									},
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
					*kargoapi.VerificationInfo,
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
				getAvailableFreightByOriginFn: func(
					context.Context, *kargoapi.Stage, bool,
				) (map[string][]kargoapi.Freight, error) {
					return map[string][]kargoapi.Freight{
						testOrigin.String(): {
							{
								ObjectMeta: metav1.ObjectMeta{
									Name: "fake-freight-id",
								},
							},
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
					newStatus.FreightHistory.Current().VerificationHistory.Current(),
				)

				// Two events should have been recorded:
				require.Len(t, recorder.Events, 2)

				// Successful verification should have been recorded as an event
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

				// Auto-promotion should have been recorded as an event
				event = <-recorder.Events
				require.Equal(t, kargoapi.EventReasonPromotionCreated, event.Reason)
			},
		},

		{
			name: "success with multiple Freight requests",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight:    []kargoapi.FreightRequest{{}, {}},
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
				Status: kargoapi.StageStatus{
					Phase: kargoapi.StagePhaseSteady,
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Origin: testOrigin,
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
				verifyFreightInStageFn: func(context.Context, string, string, string) (bool, error) {
					return false, nil
				},
				isAutoPromotionPermittedFn: func(
					context.Context,
					string,
					string,
				) (bool, error) {
					return true, nil
				},
				getAvailableFreightByOriginFn: func(
					context.Context, *kargoapi.Stage, bool,
				) (map[string][]kargoapi.Freight, error) {
					return map[string][]kargoapi.Freight{
						testOrigin.String(): {
							{
								ObjectMeta: metav1.ObjectMeta{
									CreationTimestamp: metav1.NewTime(fakeTime),
									Name:              "fake-freight-1",
									Namespace:         "fake-namespace",
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									CreationTimestamp: metav1.NewTime(fakeTime.Add(time.Hour)),
									Name:              "fake-freight-2",
									Namespace:         "fake-namespace",
								},
							},
						},
						(&kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "fake-warehouse-2",
						}).String(): {
							{
								ObjectMeta: metav1.ObjectMeta{
									CreationTimestamp: metav1.NewTime(fakeTime.Add(-1 * time.Hour)),
									Name:              "fake-freight-3",
									Namespace:         "fake-namespace",
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									CreationTimestamp: metav1.NewTime(fakeTime),
									Name:              "fake-freight-4",
									Namespace:         "fake-namespace",
								},
							},
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

				require.Equal(t, kargoapi.StagePhaseSteady, newStatus.Phase)

				// Two events should have been recorded:
				require.Len(t, recorder.Events, 2)

				// Two auto-promotions should have been recorded as events
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonPromotionCreated, event.Reason)

				event = <-recorder.Events
				require.Equal(t, kargoapi.EventReasonPromotionCreated, event.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(10)
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

	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}

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
				require.Equal(t, &kargoapi.PromotionReference{
					Name: "fake-promotion",
					Freight: &kargoapi.FreightReference{
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
				CurrentPromotion: &kargoapi.PromotionReference{
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
									Name:   "fake-freight-1",
									Origin: testOrigin,
								},
								FreightCollection: &kargoapi.FreightCollection{
									Freight: map[string]kargoapi.FreightReference{
										testOrigin.String(): {
											Name:   "fake-freight-1",
											Origin: testOrigin,
										},
									},
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
									Name:   "fake-freight-3",
									Origin: testOrigin,
								},
								FreightCollection: &kargoapi.FreightCollection{
									Freight: map[string]kargoapi.FreightReference{
										testOrigin.String(): {
											Name:   "fake-freight-3",
											Origin: testOrigin,
										},
									},
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
									Name:   "fake-freight-2",
									Origin: testOrigin,
								},
								FreightCollection: &kargoapi.FreightCollection{
									Freight: map[string]kargoapi.FreightReference{
										testOrigin.String(): {
											Name:   "fake-freight-2",
											Origin: testOrigin,
										},
									},
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

				status.LastPromotion.FinishedAt = nil

				// Last Promotion should be the latest Terminated Promotion
				require.Equal(t, &kargoapi.PromotionReference{
					Name: "fake-promotion." + ulidOneMinuteAgo.String(),
					Status: &kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseErrored,
						Freight: &kargoapi.FreightReference{
							Name:   "fake-freight-3",
							Origin: testOrigin,
						},
						FreightCollection: &kargoapi.FreightCollection{
							Freight: map[string]kargoapi.FreightReference{
								testOrigin.String(): {
									Name:   "fake-freight-3",
									Origin: testOrigin,
								},
							},
						},
					},
					Freight: &kargoapi.FreightReference{
						Name:   "fake-freight-3",
						Origin: testOrigin,
					},
				}, status.LastPromotion)

				current := status.FreightHistory.Current()
				require.NotNil(t, current)
				require.Contains(t, current.Freight, testOrigin.String())

				// Current Freight should be the Freight of the last Succeeded Promotion
				require.Equal(
					t,
					kargoapi.FreightReference{
						Name:   "fake-freight-1",
						Origin: testOrigin,
					},
					current.Freight[testOrigin.String()],
				)
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
				LastPromotion: &kargoapi.PromotionReference{
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
				require.Equal(t, &kargoapi.PromotionReference{
					Name: "fake-promotion." + ulidOneHourAgo.String(),
					Status: &kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseFailed,
					},
				}, status.LastPromotion)

				require.Len(t, status.FreightHistory, 0)
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

func TestGetAvailableFreight(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name       string
		reqs       []kargoapi.FreightRequest
		reconciler *reconciler
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "error getting getting Freight from Warehouse",
			reqs: []kargoapi.FreightRequest{
				{
					Origin: testOrigin,
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
			},
			reconciler: &reconciler{
				listFreightFn: func(context.Context, client.ObjectList, ...client.ListOption) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error listing Freight from Warehouse")
			},
		},
		{
			name: "error getting Freight verified in upstream Stages",
			reqs: []kargoapi.FreightRequest{
				{
					Origin: testOrigin,
					Sources: kargoapi.FreightSources{
						Stages: []string{"fake-stage"},
					},
				},
			},
			reconciler: &reconciler{
				listFreightFn: func(context.Context, client.ObjectList, ...client.ListOption) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error listing Freight verified in Stage")
			},
		},
		{
			name: "error getting Freight approved for Stage",
			reconciler: &reconciler{
				listFreightFn: func(context.Context, client.ObjectList, ...client.ListOption) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error listing Freight approved for Stage")
			},
		},
		{
			name: "no available Freight found",
			reconciler: &reconciler{
				listFreightFn: func(context.Context, client.ObjectList, ...client.ListOption) error {
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
			reqs: []kargoapi.FreightRequest{
				{
					Origin: testOrigin,
					Sources: kargoapi.FreightSources{
						Direct: true,
						Stages: []string{"fake-upstream-stage"},
					},
				},
			},
			reconciler: &reconciler{
				// This should end up called multiple times, but we expect the results
				// to be de-duped
				listFreightFn: func(_ context.Context, objList client.ObjectList, _ ...client.ListOption) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{}}
					return nil
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.reconciler.getAvailableFreight(
				context.Background(),
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
						Name:      "fake-stage",
					},
					Spec: kargoapi.StageSpec{
						RequestedFreight: testCase.reqs,
					},
				},
				true,
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGetAvailableFreightByOrigin(t *testing.T) {
	testCases := []struct {
		name            string
		stage           *kargoapi.Stage
		includeApproved bool
		objects         []client.Object
		interceptor     interceptor.Funcs
		assertions      func(*testing.T, map[string][]kargoapi.Freight, error)
	}{
		{
			name: "Freight directly from Warehouse",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-1",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-2",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
				},
				// Should not be included: different Warehouse
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "other-fake-warehouse",
					},
				},
			},
			assertions: func(t *testing.T, result map[string][]kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, result, 1)

				const expectOrigin = "Warehouse/fake-warehouse"
				freight, ok := result[expectOrigin]
				require.True(t, ok)
				require.Len(t, freight, 2)

				var found []string
				for _, f := range freight {
					found = append(found, f.Name)
				}
				require.Contains(t, found, "fake-freight-1")
				require.Contains(t, found, "fake-freight-2")
			},
		},
		{
			name: "Freight from upstream Stages",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Stages: []string{"fake-upstream-1", "fake-upstream-2"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-1",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"fake-upstream-1": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-2",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"fake-upstream-2": {},
						},
					},
				},
				// Should not be included: not verified in any upstream Stages
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-3",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
				},
				// Should not be included: different Stage
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-4",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"other-fake-upstream": {},
						},
					},
				},
			},
			assertions: func(t *testing.T, result map[string][]kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, result, 1)

				const expectOrigin = "Warehouse/fake-warehouse"
				freight, ok := result[expectOrigin]
				require.True(t, ok)
				require.Len(t, freight, 2)

				var found []string
				for _, f := range freight {
					found = append(found, f.Name)
				}
				require.Contains(t, found, "fake-freight-1")
				require.Contains(t, found, "fake-freight-2")
			},
		},
		{
			name: "approved Freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-1",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"fake-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-2",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"fake-stage": {},
						},
					},
				},
				// Should not be included: not approved for this Stage
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-3",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"other-fake-stage": {},
						},
					},
				},
			},
			includeApproved: true,
			assertions: func(t *testing.T, result map[string][]kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, result, 1)

				const expectOrigin = "Warehouse/fake-warehouse"
				freight, ok := result[expectOrigin]
				require.True(t, ok)
				require.Len(t, freight, 2)

				var found []string
				for _, f := range freight {
					found = append(found, f.Name)
				}
				require.Contains(t, found, "fake-freight-1")
			},
		},
		{
			name: "deduplicates Freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Stages: []string{"fake-stage-1", "fake-stage-2"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-1",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"fake-stage-1": {},
							"fake-stage-2": {},
						},
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"fake-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-2",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"fake-stage-1": {},
							"fake-stage-2": {},
						},
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"fake-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-3",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"fake-stage": {},
						},
					},
				},
			},
			includeApproved: true,
			assertions: func(t *testing.T, result map[string][]kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, result, 1)

				const expectOrigin = "Warehouse/fake-warehouse"
				freight, ok := result[expectOrigin]
				require.True(t, ok)
				require.Len(t, freight, 3)

				var found []string
				for _, f := range freight {
					found = append(found, f.Name)
				}
				require.Contains(t, found, "fake-freight-1")
				require.Contains(t, found, "fake-freight-2")
				require.Contains(t, found, "fake-freight-3")
			},
		},
		{
			name: "error listing direct Freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, result map[string][]kargoapi.Freight, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error listing Freight")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, result)
			},
		},
		{
			name: "error listing Freight verified in upstream Stages",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Stages: []string{"fake-upstream-stage"},
							},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				List: func(
					ctx context.Context,
					c client.WithWatch,
					l client.ObjectList,
					opts ...client.ListOption,
				) error {
					lo := &client.ListOptions{}
					lo.ApplyOptions(opts)

					if strings.Contains(
						lo.FieldSelector.String(),
						fmt.Sprintf("%s=%s", kubeclient.FreightByVerifiedStagesIndexField, "fake-upstream-stage"),
					) {
						return fmt.Errorf("something went wrong")
					}

					return c.List(ctx, l, opts...)
				},
			},
			assertions: func(t *testing.T, result map[string][]kargoapi.Freight, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error listing Freight verified in Stage \"fake-upstream-stage\"")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, result)
			},
		},
		{
			name: "error listing Freight approved for Stage",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
						},
					},
				},
			},
			includeApproved: true,
			interceptor: interceptor.Funcs{
				List: func(
					ctx context.Context,
					c client.WithWatch,
					l client.ObjectList,
					opts ...client.ListOption,
				) error {
					lo := &client.ListOptions{}
					lo.ApplyOptions(opts)

					if strings.Contains(
						lo.FieldSelector.String(),
						fmt.Sprintf("%s=%s", kubeclient.FreightApprovedForStagesIndexField, "fake-stage"),
					) {
						return fmt.Errorf("something went wrong")
					}

					return c.List(ctx, l, opts...)
				},
			},
			assertions: func(t *testing.T, result map[string][]kargoapi.Freight, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error listing Freight approved for Stage")
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, result)
			},
		},
		{
			name: "listing direct Freight skips other sources",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-stage",
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
								Stages: []string{"fake-upstream-stage"},
							},
						},
					},
				},
			},
			includeApproved: true,
			interceptor: interceptor.Funcs{
				List: func(
					ctx context.Context,
					c client.WithWatch,
					l client.ObjectList,
					opts ...client.ListOption,
				) error {
					lo := &client.ListOptions{}
					lo.ApplyOptions(opts)

					if strings.Contains(lo.FieldSelector.String(), kubeclient.FreightApprovedForStagesIndexField) {
						return fmt.Errorf("something went wrong")
					}

					if strings.Contains(lo.FieldSelector.String(), kubeclient.FreightByVerifiedStagesIndexField) {
						return fmt.Errorf("something went wrong")
					}

					return c.List(ctx, l, opts...)
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-freight-1",
						Namespace: "fake-namespace",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
				},
			},
			assertions: func(t *testing.T, result map[string][]kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, result, 1)

				const expectOrigin = "Warehouse/fake-warehouse"
				freight, ok := result[expectOrigin]
				require.True(t, ok)
				require.Len(t, freight, 1)
			},
		},
	}

	s := runtime.NewScheme()
	assert.NoError(t, kargoapi.AddToScheme(s))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(s).
				WithIndex(
					&kargoapi.Freight{},
					kubeclient.FreightByWarehouseIndexField,
					kubeclient.FreightByWarehouseIndexer,
				).
				WithIndex(
					&kargoapi.Freight{},
					kubeclient.FreightByVerifiedStagesIndexField,
					kubeclient.FreightByVerifiedStagesIndexer,
				).
				WithIndex(
					&kargoapi.Freight{},
					kubeclient.FreightApprovedForStagesIndexField,
					kubeclient.FreightApprovedForStagesIndexer,
				).
				WithInterceptorFuncs(tc.interceptor).
				WithObjects(tc.objects...).
				Build()

			r := &reconciler{
				listFreightFn: c.List,
			}

			result, err := r.getAvailableFreightByOrigin(context.Background(), tc.stage, tc.includeApproved)
			tc.assertions(t, result, err)
		})
	}
}

func TestBuildFreightSummary(t *testing.T) {
	testCases := []struct {
		name            string
		requested       int
		currentFreight  *kargoapi.FreightCollection
		expectedSummary string
	}{
		{
			name:            "requested 1, got none",
			requested:       1,
			expectedSummary: "0/1 Fulfilled",
		},
		{
			name:      "requested 1, got 1",
			requested: 1,
			currentFreight: &kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"Warehouse/fake-warehouse": {
						Name: "fake-freight",
					},
				},
			},
			expectedSummary: "fake-freight",
		},
		{
			name:            "requested multiple, got none",
			requested:       2,
			expectedSummary: "0/2 Fulfilled",
		},
		{
			name:      "requested multiple, got some",
			requested: 2,
			currentFreight: &kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"Warehouse/fake-warehouse": {
						Name: "fake-freight",
					},
				},
			},
			expectedSummary: "1/2 Fulfilled",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expectedSummary,
				buildFreightSummary(testCase.requested, testCase.currentFreight),
			)
		})
	}
}

func fakeNow() time.Time {
	return fakeTime
}
