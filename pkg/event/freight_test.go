package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestFreightVerification_MarshalAnnotationsTo(t *testing.T) {
	testCases := map[string]struct {
		verification FreightVerification
		expected     map[string]string
	}{
		"complete verification": {
			verification: FreightVerification{
				StartTime:                    ptr.To(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				FinishTime:                   ptr.To(time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)),
				AnalysisRunName:              ptr.To("test-analysis"),
				AnalysisTriggeredByPromotion: ptr.To("test-promotion"),
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventVerificationStartTime:  "2024-01-01T10:00:00Z",
				kargoapi.AnnotationKeyEventVerificationFinishTime: "2024-01-01T11:00:00Z",
				kargoapi.AnnotationKeyEventAnalysisRunName:        "test-analysis",
				kargoapi.AnnotationKeyEventPromotionName:          "test-promotion",
			},
		},
		"minimal verification": {
			verification: FreightVerification{},
			expected:     map[string]string{},
		},
		"partial verification": {
			verification: FreightVerification{
				StartTime:       ptr.To(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				AnalysisRunName: ptr.To("test-analysis"),
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventVerificationStartTime: "2024-01-01T10:00:00Z",
				kargoapi.AnnotationKeyEventAnalysisRunName:       "test-analysis",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			annotations := make(map[string]string)
			tc.verification.MarshalAnnotationsTo(annotations)
			require.Equal(t, tc.expected, annotations)
		})
	}
}

func TestNewFreightVerification(t *testing.T) {
	testCases := map[string]struct {
		verificationInfo *kargoapi.VerificationInfo
		expected         FreightVerification
	}{
		"complete verification info": {
			verificationInfo: &kargoapi.VerificationInfo{
				StartTime:  &metav1.Time{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
				FinishTime: &metav1.Time{Time: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
				AnalysisRun: &kargoapi.AnalysisRunReference{
					Name: "test-analysis",
				},
			},
			expected: FreightVerification{
				StartTime:       ptr.To(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				FinishTime:      ptr.To(time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)),
				AnalysisRunName: ptr.To("test-analysis"),
			},
		},
		"nil verification info": {
			verificationInfo: nil,
			expected:         FreightVerification{},
		},
		"partial verification info": {
			verificationInfo: &kargoapi.VerificationInfo{
				StartTime: &metav1.Time{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
			},
			expected: FreightVerification{
				StartTime: ptr.To(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := NewFreightVerification(tc.verificationInfo)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFreightVerificationSucceeded(t *testing.T) {
	evt := &FreightVerificationSucceeded{}
	require.Equal(t, kargoapi.EventTypeFreightVerificationSucceeded, evt.Type())
}

func TestFreightVerificationFailed(t *testing.T) {
	evt := &FreightVerificationFailed{}
	require.Equal(t, kargoapi.EventTypeFreightVerificationFailed, evt.Type())
}

func TestFreightVerificationErrored(t *testing.T) {
	evt := &FreightVerificationErrored{}
	require.Equal(t, kargoapi.EventTypeFreightVerificationErrored, evt.Type())
}

func TestFreightVerificationAborted(t *testing.T) {
	evt := &FreightVerificationAborted{}
	require.Equal(t, kargoapi.EventTypeFreightVerificationAborted, evt.Type())
}

func TestFreightVerificationInconclusive(t *testing.T) {
	evt := &FreightVerificationInconclusive{}
	require.Equal(t, kargoapi.EventTypeFreightVerificationInconclusive, evt.Type())
}

func TestFreightVerificationUnknown(t *testing.T) {
	evt := &FreightVerificationUnknown{}
	require.Equal(t, kargoapi.EventTypeFreightVerificationUnknown, evt.Type())
}

func TestFreightApproved(t *testing.T) {
	evt := &FreightApproved{}
	require.Equal(t, kargoapi.EventTypeFreightApproved, evt.Type())
}

func TestNewFreightCommon(t *testing.T) {
	testCases := map[string]struct {
		message         string
		actor           string
		stageName       string
		freight         *kargoapi.Freight
		expectedCommon  Common
		expectedFreight Freight
	}{
		"complete freight": {
			message:   "test message",
			actor:     "test-actor",
			stageName: "test-stage",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-freight",
					Namespace:         "test-project",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
				Alias: "v1.0.0",
			},
			expectedCommon: Common{
				Project: "test-project",
				Message: "test message",
				Actor:   ptr.To("test-actor"),
			},
			expectedFreight: Freight{
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Name:       "test-freight",
				StageName:  "test-stage",
				Alias:      ptr.To("v1.0.0"),
			},
		},
		"nil freight": {
			message:   "test message",
			actor:     "test-actor",
			stageName: "test-stage",
			freight:   nil,
			expectedCommon: Common{
				Message: "test message",
				Actor:   ptr.To("test-actor"),
			},
			expectedFreight: Freight{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			common, freight := NewFreightCommon(tc.message, tc.actor, tc.stageName, tc.freight)
			require.Equal(t, tc.expectedCommon, common)
			require.Equal(t, tc.expectedFreight, freight)
		})
	}
}

func TestFreightVerificationConstructors(t *testing.T) {
	// NOTE(thomastaylor312): I'm including these for now as we might need to test more edge cases
	// if we expand events. If this test isn't adding any value in the future, we can remove
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-project",
			CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	verification := &kargoapi.VerificationInfo{
		Message:    "Verification completed",
		StartTime:  &metav1.Time{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		FinishTime: &metav1.Time{Time: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
	}

	testCases := map[string]struct {
		constructor  func() any
		expectedType kargoapi.EventType
	}{
		"succeeded": {
			constructor: func() any {
				return NewFreightVerificationSucceeded("test-actor", "test-stage", freight, verification)
			},
			expectedType: kargoapi.EventTypeFreightVerificationSucceeded,
		},
		"failed": {
			constructor: func() any {
				return NewFreightVerificationFailed("test-actor", "test-stage", freight, verification)
			},
			expectedType: kargoapi.EventTypeFreightVerificationFailed,
		},
		"errored": {
			constructor: func() any {
				return NewFreightVerificationErrored("test-actor", "test-stage", freight, verification)
			},
			expectedType: kargoapi.EventTypeFreightVerificationErrored,
		},
		"aborted": {
			constructor: func() any {
				return NewFreightVerificationAborted("test-actor", "test-stage", freight, verification)
			},
			expectedType: kargoapi.EventTypeFreightVerificationAborted,
		},
		"inconclusive": {
			constructor: func() any {
				return NewFreightVerificationInconclusive("test-actor", "test-stage", freight, verification)
			},
			expectedType: kargoapi.EventTypeFreightVerificationInconclusive,
		},
		"unknown": {
			constructor: func() any {
				return NewFreightVerificationUnknown("test-actor", "test-stage", freight, verification)
			},
			expectedType: kargoapi.EventTypeFreightVerificationUnknown,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			event := tc.constructor()

			// Verify the event implements Meta and has correct type
			eventMeta, ok := event.(Meta)
			require.True(t, ok)
			require.Equal(t, tc.expectedType, eventMeta.Type())

			// Verify common fields are set correctly
			require.Equal(t, "test-project", eventMeta.GetProject())
			require.Equal(t, "test-freight", eventMeta.GetName())
			require.Equal(t, "Freight", eventMeta.Kind())
		})
	}
}

func TestNewFreightApproved(t *testing.T) {
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-project",
			CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	event := NewFreightApproved("Freight approved", "test-actor", "test-stage", freight)

	require.Equal(t, kargoapi.EventTypeFreightApproved, event.Type())
	require.Equal(t, "test-project", event.GetProject())
	require.Equal(t, "test-freight", event.GetName())
	require.Equal(t, "Freight", event.Kind())
	require.Equal(t, "Freight approved", event.GetMessage())
}

func TestFreightEventMarshalAnnotations(t *testing.T) {
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-project",
			CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Alias: "v1.0.0",
	}
	verification := &kargoapi.VerificationInfo{
		StartTime:  &metav1.Time{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		FinishTime: &metav1.Time{Time: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
	}

	testCases := map[string]struct {
		event    AnnotationMarshaler
		expected map[string]string
	}{
		"freight verification succeeded": {
			event: NewFreightVerificationSucceeded("test-actor", "test-stage", freight, verification),
			expected: map[string]string{
				kargoapi.AnnotationKeyEventProject:                "test-project",
				kargoapi.AnnotationKeyEventActor:                  "test-actor",
				kargoapi.AnnotationKeyEventFreightName:            "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime:      "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:              "test-stage",
				kargoapi.AnnotationKeyEventFreightAlias:           "v1.0.0",
				kargoapi.AnnotationKeyEventVerificationStartTime:  "2024-01-01T10:00:00Z",
				kargoapi.AnnotationKeyEventVerificationFinishTime: "2024-01-01T11:00:00Z",
			},
		},
		"freight approved": {
			event: NewFreightApproved("Freight approved", "test-actor", "test-stage", freight),
			expected: map[string]string{
				kargoapi.AnnotationKeyEventProject:           "test-project",
				kargoapi.AnnotationKeyEventActor:             "test-actor",
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
				kargoapi.AnnotationKeyEventFreightAlias:      "v1.0.0",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			annotations := tc.event.MarshalAnnotations()
			require.Equal(t, tc.expected, annotations)
		})
	}
}

func TestUnmarshalFreightVerificationAnnotations(t *testing.T) {
	testCases := map[string]struct {
		annotations  map[string]string
		expected     FreightVerification
		expectError  bool
		errorMessage string
	}{
		"complete annotations": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventVerificationStartTime:  "2024-01-01T10:00:00Z",
				kargoapi.AnnotationKeyEventVerificationFinishTime: "2024-01-01T11:00:00Z",
				kargoapi.AnnotationKeyEventAnalysisRunName:        "test-analysis",
				kargoapi.AnnotationKeyEventPromotionName:          "test-promotion",
			},
			expected: FreightVerification{
				StartTime:                    ptr.To(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				FinishTime:                   ptr.To(time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)),
				AnalysisRunName:              ptr.To("test-analysis"),
				AnalysisTriggeredByPromotion: ptr.To("test-promotion"),
			},
		},
		"minimal annotations": {
			annotations: map[string]string{},
			expected:    FreightVerification{},
		},
		"invalid start time": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventVerificationStartTime: "invalid-time",
			},
			expectError:  true,
			errorMessage: "failed to parse verification start time",
		},
		"invalid finish time": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventVerificationFinishTime: "invalid-time",
			},
			expectError:  true,
			errorMessage: "failed to parse verification finish time",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := UnmarshalFreightVerificationAnnotations(tc.annotations)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMessage)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFreightEventUnmarshalAnnotations(t *testing.T) {
	testCases := map[string]struct {
		annotations   map[string]string
		unmarshalFunc func(map[string]string) (Meta, error)
		expectedType  Meta
		expectError   bool
		errorMessage  string
	}{
		"freight verification succeeded": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:                "test-project",
				kargoapi.AnnotationKeyEventFreightName:            "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime:      "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:              "test-stage",
				kargoapi.AnnotationKeyEventVerificationStartTime:  "2024-01-01T10:00:00Z",
				kargoapi.AnnotationKeyEventVerificationFinishTime: "2024-01-01T11:00:00Z",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalFreightVerificationSucceededAnnotations("event-id", annotations)
			},
			expectedType: &FreightVerificationSucceeded{
				Common: Common{
					Project: "test-project",
					ID:      "event-id",
				},
				Freight: Freight{
					Name:       "test-freight",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				FreightVerification: FreightVerification{
					StartTime:  ptr.To(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
					FinishTime: ptr.To(time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)),
				},
			},
		},
		"freight approved": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject:           "test-project",
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalFreightApprovedAnnotations("event-id", annotations)
			},
			expectedType: &FreightApproved{
				Common: Common{
					Project: "test-project",
					ID:      "event-id",
				},
				Freight: Freight{
					Name:       "test-freight",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		"invalid freight annotations": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "invalid-time",
			},
			unmarshalFunc: func(annotations map[string]string) (Meta, error) {
				return UnmarshalFreightApprovedAnnotations("event-id", annotations)
			},
			expectError:  true,
			errorMessage: "failed to parse freight create time",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := tc.unmarshalFunc(tc.annotations)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMessage)
				return
			}

			require.NoError(t, err)

			// Deep-compare the full struct via Meta interface
			require.Equal(t, tc.expectedType, result, "oh noes, types don't match!")
		})
	}
}
