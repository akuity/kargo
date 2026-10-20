package verification

import (
	"context"
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
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
)

func TestVerifier_recordFreightVerificationEvent(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := metav1.Now()
	startTime := metav1.NewTime(now.Add(-1 * time.Hour))
	finishTime := metav1.NewTime(now.Add(-30 * time.Minute))

	baseStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stage",
			Namespace: "test-project",
		},
	}

	baseFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-project",
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
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationSucceeded), event.Reason)
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
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationFailed), event.Reason)
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
					Namespace: "test-project",
				},
			},
			objects: []client.Object{
				baseFreight,
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "test-project",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyPromotion: "test-promotion",
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
					Namespace: "test-project",
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
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationErrored), event.Reason)
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
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationAborted), event.Reason)
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
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationInconclusive), event.Reason)
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
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationUnknown), event.Reason)
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
			ver := Verifier{
				client:      c,
				eventSender: k8sevent.NewEventSender(recorder),
				analysisRunners: map[kargoapi.AnalysisRunGVK]AnalysisRunner{
					kargoapi.AnalysisRunGVKRun: &analysisRunnerRollouts{
						client:                       c,
						rolloutsControllerInstanceID: "test-instance",
					},
				},
			}

			ver.recordFreightVerificationEvent(tt.stage, tt.freightRef, tt.vi)
			tt.assertions(t, recorder)
		})
	}
}

func TestVerifier_startVerification(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	startTime := time.Now()
	endTime := startTime.Add(5 * time.Minute)
	fixedEndTime := func() time.Time { return endTime }

	tests := []struct {
		name             string
		stage            *kargoapi.Stage
		freightCol       kargoapi.FreightCollection
		req              *kargoapi.VerificationRequest
		objects          []client.Object
		credsDB          credentials.Database
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
				assert.Equal(t, startTime, vi.StartTime.Time)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
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
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
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
				// StartTime is the injected reconciliation time; FinishTime is
				// the injected end time stamped when the result is recorded.
				require.NotNil(t, vi.StartTime)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, startTime.Unix(), vi.StartTime.Unix())
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
			},
		},
		{
			name: "finds existing analysis run with stage name exceeding max label length",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
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
							kargoapi.LabelKeyStage:             "this-is-a-very-long-stage-name-that-exceeds-the-label-1c0a17e1",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyStage: "this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated", // nolint:lll
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
				require.NotNil(t, vi.StartTime)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, startTime.Unix(), vi.StartTime.Unix())
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
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
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))

				// Verify stage label is not shortened since stage name is short
				assert.Equal(t, "test-stage", ar.Labels[kargoapi.LabelKeyStage])

				// Verify no annotation is added since stage name doesn't need shortening
				_, hasAnnotation := ar.Annotations[kargoapi.AnnotationKeyStage]
				assert.False(t, hasAnnotation, "Stage annotation should not be present when stage name doesn't need shortening")
			},
		},
		{
			name: "creates new analysis run with stage name exceeding max label length",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
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
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))

				// Verify stage label was truncated correctly
				assert.Equal(
					t,
					"this-is-a-very-long-stage-name-that-exceeds-the-label-1c0a17e1",
					ar.Labels[kargoapi.LabelKeyStage],
				)

				// Verify annotation contains the full stage name
				fullStageName, hasAnnotation := ar.Annotations[kargoapi.AnnotationKeyStage]
				assert.True(t, hasAnnotation)
				assert.Equal(
					t,
					"this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
					fullStageName,
				)
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

				// Verify promotion annotation was added
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))
				assert.Equal(t, "test-promotion", ar.Annotations[kargoapi.AnnotationKeyPromotion])

				// Verify no stage annotation is added since stage name doesn't need shortening
				_, hasStageAnnotation := ar.Annotations[kargoapi.AnnotationKeyStage]
				assert.False(t, hasStageAnnotation)
			},
		},
		{
			name: "handles reverification with control plane actor and long stage name",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
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

				// Verify analysis run was created
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))

				// Verify both promotion and stage annotations are present
				assert.Equal(t, "test-promotion", ar.Annotations[kargoapi.AnnotationKeyPromotion])
				fullStageName, hasStageAnnotation := ar.Annotations[kargoapi.AnnotationKeyStage]
				assert.True(t, hasStageAnnotation)
				assert.Equal(
					t,
					"this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
					fullStageName,
				)

				// Verify stage label was truncated correctly
				assert.Equal(t,
					"this-is-a-very-long-stage-name-that-exceeds-the-label-1c0a17e1",
					ar.Labels[kargoapi.LabelKeyStage],
				)
			},
		},
		{
			name: "resolves repoCredentials() in verification arguments",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
							{Name: "test-template"},
						},
						Args: []kargoapi.AnalysisRunArgument{
							{
								Name: "token",
								Value: "${{ repoCredentials(" +
									"'https://github.com/example/repo.git', 'git'" +
									").Password }}",
							},
						},
					},
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
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "fake-project",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Args: []rolloutsapi.Argument{{Name: "token"}},
					},
				},
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (*credentials.Credentials, error) {
					return &credentials.Credentials{Password: "s3cr3t"}, nil
				},
			},
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhasePending, vi.Phase)
				require.NotNil(t, vi.AnalysisRun)

				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))

				require.Len(t, ar.Spec.Args, 1)
				assert.Equal(t, "token", ar.Spec.Args[0].Name)
				require.NotNil(t, ar.Spec.Args[0].Value)
				assert.Equal(t, "s3cr3t", *ar.Spec.Args[0].Value)
			},
		},
		{
			name: "surfaces error when repoCredentials() is unavailable",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
							{Name: "test-template"},
						},
						Args: []kargoapi.AnalysisRunArgument{
							{
								Name: "token",
								Value: "${{ repoCredentials(" +
									"'https://github.com/example/repo.git', 'git'" +
									").Password }}",
							},
						},
					},
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
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "fake-project",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Args: []rolloutsapi.Argument{{Name: "token"}},
					},
				},
			},
			// credsDB intentionally left nil.
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "error building AnalysisRun")
				assert.Contains(t, vi.Message, "repoCredentials is not available")
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

			ver := Verifier{
				client: c,

				rolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				analysisRunners: map[kargoapi.AnalysisRunGVK]AnalysisRunner{
					kargoapi.AnalysisRunGVKRun: &analysisRunnerRollouts{
						client:                       c,
						rolloutsControllerInstanceID: "test-instance",
						credentialsDB:                tt.credsDB,
					},
				},
			}

			vi, err := ver.startVerification(t.Context(), tt.stage, tt.freightCol, tt.req, startTime, fixedEndTime)
			tt.assertions(t, c, vi, err)
		})
	}
}

func TestVerifier_getVerificationResult(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()
	endTime := now.Add(5 * time.Minute)
	fixedEndTime := func() time.Time { return endTime }

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
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
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
						Phase:       rolloutsapi.AnalysisPhaseSuccessful,
						Message:     "Analysis completed successfully",
						CompletedAt: &metav1.Time{Time: endTime},
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: endTime},
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
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
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
						Phase:       "Failed",
						Message:     "Something went wrong",
						CompletedAt: &metav1.Time{Time: endTime},
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: endTime},
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
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
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
						Phase:       rolloutsapi.AnalysisPhaseError,
						Message:     "Something went wrong",
						CompletedAt: &metav1.Time{Time: endTime},
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: endTime},
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
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
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

			ver := Verifier{
				client:                     c,
				rolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				analysisRunners: map[kargoapi.AnalysisRunGVK]AnalysisRunner{
					kargoapi.AnalysisRunGVKRun: &analysisRunnerRollouts{
						client: c,
						backoffCfg: wait.Backoff{
							Duration: 1 * time.Second,
							Factor:   2,
							Steps:    2,
							Cap:      1 * time.Second,
							Jitter:   0.1,
						},
					},
				},
			}

			vi, err := ver.getVerificationResult(t.Context(), tt.freight, fixedEndTime)
			tt.assertions(t, vi, err)
		})
	}
}

func TestVerifier_abortVerification(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()
	endTime := now.Add(5 * time.Minute)
	fixedEndTime := func() time.Time { return endTime }

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
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
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
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
				assert.Equal(t, "test-analysis", vi.AnalysisRun.Name)

				// Verify analysis run was patched with terminate = true
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
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
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
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

			ver := Verifier{
				client:                     c,
				rolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				analysisRunners: map[kargoapi.AnalysisRunGVK]AnalysisRunner{
					kargoapi.AnalysisRunGVKRun: &analysisRunnerRollouts{
						client: c,
					},
				},
			}

			vi, err := ver.abortVerification(t.Context(), tt.freightCol, tt.req, fixedEndTime)
			tt.assertions(t, c, vi, err)
		})
	}
}

func TestVerifier_findExistingAnalysisRun(t *testing.T) {
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
		assertions   func(*testing.T, *AnalysisRun, error)
	}{
		{
			name: "no analysis runs found",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			assertions: func(t *testing.T, ar *AnalysisRun, err error) {
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
			assertions: func(t *testing.T, ar *AnalysisRun, err error) {
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
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
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
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase: "Failed",
					},
				},
			},
			assertions: func(t *testing.T, ar *AnalysisRun, err error) {
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
							kargoapi.LabelKeyStage:             "other-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-stage-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *AnalysisRun, err error) {
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
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "other-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-freight-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *AnalysisRun, err error) {
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
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-namespace-analysis",
						Namespace:         "test-namespace",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *AnalysisRun, err error) {
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
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *AnalysisRun, err error) {
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

			ver := Verifier{
				client: c,
				analysisRunners: map[kargoapi.AnalysisRunGVK]AnalysisRunner{
					kargoapi.AnalysisRunGVKRun: &analysisRunnerRollouts{
						client: c,
					},
				},
			}

			ar, err := ver.findExistingAnalysisRun(t.Context(), tt.stage, tt.freightColID)
			tt.assertions(t, ar, err)
		})
	}
}
