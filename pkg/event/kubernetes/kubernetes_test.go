package kubernetes

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
)

func TestFromKubernetesEvent(t *testing.T) {
	testCases := map[string]struct {
		k8sEvent        corev1.Event
		expectedType    kargoapi.EventType
		expectedData    any
		expectError     bool
		errorMessage    string
		extraValidation func(*testing.T, event.Meta)
	}{
		"promotion succeeded event": {
			k8sEvent: corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					UID: "test-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyEventProject:             "test-project",
						kargoapi.AnnotationKeyEventActor:               "test-actor",
						kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
						kargoapi.AnnotationKeyEventStageName:           "test-stage",
						kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T00:00:00Z",
						kargoapi.AnnotationKeyEventFreightName:         "test-freight",
						kargoapi.AnnotationKeyEventFreightCreateTime:   "2024-01-01T00:00:00Z",
						kargoapi.AnnotationKeyEventApplications:        `[{"namespace":"argocd","name":"app1"}]`,
					},
				},
				Reason:  string(kargoapi.EventTypePromotionSucceeded),
				Message: "Promotion succeeded",
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Promotion",
					Name:      "test-promotion",
					Namespace: "test-project",
				},
				LastTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			expectedType: kargoapi.EventTypePromotionSucceeded,
		},
		"promotion failed event": {
			k8sEvent: corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					UID: "test-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyEventProject:             "test-project",
						kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
						kargoapi.AnnotationKeyEventStageName:           "test-stage",
						kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T00:00:00Z",
					},
				},
				Reason:  string(kargoapi.EventTypePromotionFailed),
				Message: "Promotion failed",
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Promotion",
					Name:      "test-promotion",
					Namespace: "test-project",
				},
				LastTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			expectedType: kargoapi.EventTypePromotionFailed,
		},
		"freight verification succeeded event": {
			k8sEvent: corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					UID: "test-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyEventProject:                "test-project",
						kargoapi.AnnotationKeyEventFreightName:            "test-freight",
						kargoapi.AnnotationKeyEventStageName:              "test-stage",
						kargoapi.AnnotationKeyEventFreightCreateTime:      "2024-01-01T00:00:00Z",
						kargoapi.AnnotationKeyEventVerificationStartTime:  "2024-01-01T10:00:00Z",
						kargoapi.AnnotationKeyEventVerificationFinishTime: "2024-01-01T11:00:00Z",
					},
				},
				Reason:  string(kargoapi.EventTypeFreightVerificationSucceeded),
				Message: "Verification succeeded",
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Freight",
					Name:      "test-freight",
					Namespace: "test-project",
				},
				LastTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			expectedType: kargoapi.EventTypeFreightVerificationSucceeded,
		},
		"freight approved event": {
			k8sEvent: corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					UID: "test-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyEventProject:           "test-project",
						kargoapi.AnnotationKeyEventFreightName:       "test-freight",
						kargoapi.AnnotationKeyEventStageName:         "test-stage",
						kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
					},
				},
				Reason:  string(kargoapi.EventTypeFreightApproved),
				Message: "Freight approved",
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Freight",
					Name:      "test-freight",
					Namespace: "test-project",
				},
				LastTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			expectedType: kargoapi.EventTypeFreightApproved,
		},
		"custom event type": {
			k8sEvent: corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					UID: "test-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyEventProject:                               "test-project",
						kargoapi.AnnotationKeyEventPrefix + event.AnnotationEventKeyName: "test-event",
						kargoapi.AnnotationKeyEventPrefix + event.AnnotationEventKeyKind: "CustomKind",
						kargoapi.AnnotationKeyEventPrefix + "customField":                "customValue",
						kargoapi.AnnotationKeyEventPrefix + "jsonField":                  `{"key":"value"}`,
						"non-kargo-annotation":                                           "ignored",
					},
				},
				Reason:  "CustomEventType",
				Message: "Custom event message",
				InvolvedObject: corev1.ObjectReference{
					Kind:      "CustomResource",
					Name:      "test-resource",
					Namespace: "test-project",
				},
				LastTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			expectedType: "CustomEventType",
			extraValidation: func(t *testing.T, meta event.Meta) {
				require.Equal(t, "test-event", meta.GetName())
				require.Equal(t, "CustomKind", meta.Kind())
			},
		},
		"invalid promotion annotations": {
			k8sEvent: corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyEventPromotionCreateTime: "invalid-time",
					},
				},
				Reason: string(kargoapi.EventTypePromotionSucceeded),
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Promotion",
					Name:      "test-promotion",
					Namespace: "test-project",
				},
			},
			expectError:  true,
			errorMessage: "failed to parse promotion create time",
		},
		"invalid freight annotations": {
			k8sEvent: corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyEventFreightName:       "test-freight",
						kargoapi.AnnotationKeyEventFreightCreateTime: "invalid-time",
					},
				},
				Reason: string(kargoapi.EventTypeFreightApproved),
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Freight",
					Name:      "test-freight",
					Namespace: "test-project",
				},
			},
			expectError:  true,
			errorMessage: "failed to parse freight create time",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := FromKubernetesEvent(tc.k8sEvent)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMessage)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedType, result.Type())

			if tc.extraValidation != nil {
				tc.extraValidation(t, result)
			}
		})
	}
}

func TestFromKubernetesEvent_AllEventTypes(t *testing.T) {
	// Test all supported event types to ensure they can be converted
	eventTypes := []kargoapi.EventType{
		kargoapi.EventTypePromotionCreated,
		kargoapi.EventTypePromotionSucceeded,
		kargoapi.EventTypePromotionFailed,
		kargoapi.EventTypePromotionErrored,
		kargoapi.EventTypePromotionAborted,
		kargoapi.EventTypeFreightVerificationSucceeded,
		kargoapi.EventTypeFreightVerificationFailed,
		kargoapi.EventTypeFreightVerificationErrored,
		kargoapi.EventTypeFreightVerificationAborted,
		kargoapi.EventTypeFreightVerificationInconclusive,
		kargoapi.EventTypeFreightVerificationUnknown,
		kargoapi.EventTypeFreightApproved,
	}

	for _, eventType := range eventTypes {
		t.Run(string(eventType), func(t *testing.T) {
			var k8sEvent corev1.Event

			// Create appropriate test data based on event type
			if isPromotionEvent(eventType) {
				k8sEvent = corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						UID: "test-uid",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyEventProject:             "test-project",
							kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
							kargoapi.AnnotationKeyEventStageName:           "test-stage",
							kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T00:00:00Z",
						},
					},
					Reason:  string(eventType),
					Message: "Test message",
					InvolvedObject: corev1.ObjectReference{
						Kind:      "Promotion",
						Name:      "test-promotion",
						Namespace: "test-project",
					},
					LastTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
				}
			} else {
				annotations := map[string]string{
					kargoapi.AnnotationKeyEventProject:           "test-project",
					kargoapi.AnnotationKeyEventFreightName:       "test-freight",
					kargoapi.AnnotationKeyEventStageName:         "test-stage",
					kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				}

				// Add verification-specific annotations for verification events
				if isVerificationEvent(eventType) {
					annotations[kargoapi.AnnotationKeyEventVerificationStartTime] = "2024-01-01T10:00:00Z"
					annotations[kargoapi.AnnotationKeyEventVerificationFinishTime] = "2024-01-01T11:00:00Z"
				}

				k8sEvent = corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						UID:         "test-uid",
						Annotations: annotations,
					},
					Reason:  string(eventType),
					Message: "Test message",
					InvolvedObject: corev1.ObjectReference{
						Kind:      "Freight",
						Name:      "test-freight",
						Namespace: "test-project",
					},
					LastTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
				}
			}

			result, err := FromKubernetesEvent(k8sEvent)
			require.NoError(t, err)
			require.Equal(t, eventType, result.Type())
		})
	}
}

func TestNewEventSender(t *testing.T) {
	recorder := &mockEventRecorder{}
	sender := NewEventSender(recorder)
	require.NotNil(t, sender)
	require.Equal(t, recorder, sender.recorder)
}

func TestEventSender_Send(t *testing.T) {
	testCases := map[string]struct {
		event         event.Meta
		expectError   bool
		errorMessage  string
		expectedCalls int
	}{
		"promotion succeeded event": {
			event: &event.PromotionSucceeded{
				Common: event.Common{
					Project: "test-project",
					Message: "Promotion succeeded",
					Actor:   stringPtr("test-actor"),
				},
				Promotion: event.Promotion{
					Name:       "test-promotion",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expectedCalls: 1,
		},
		"freight approved event": {
			event: &event.FreightApproved{
				Common: event.Common{
					Project: "test-project",
					Message: "Freight approved",
					Actor:   stringPtr("test-actor"),
				},
				Freight: event.Freight{
					Name:       "test-freight",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expectedCalls: 1,
		},
		"custom event type": {
			event: &customEvent{
				Message:     "Custom event message",
				CustomField: "customValue",
				JSONField:   map[string]string{"key": "value"},
				EventType:   "CustomEventType",
				Name:        "test-resource",
				Project:     "test-project",
				ObjectKind:  "CustomResource",
				ID:          "custom-event-id",
			},
			expectedCalls: 1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			recorder := &mockEventRecorder{}
			sender := NewEventSender(recorder)

			err := sender.Send(context.Background(), tc.event)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMessage)
				require.Equal(t, 0, recorder.callCount)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedCalls, recorder.callCount)
		})
	}
}

func TestConvertToAnnotations(t *testing.T) {
	testCases := map[string]struct {
		data         event.Meta
		expected     map[string]string
		expectError  bool
		errorMessage string
	}{
		"custom event": {
			data: &customEvent{
				Message:     "Custom event message",
				CustomField: "customValue",
				JSONField:   map[string]string{"key": "value"},
				EventType:   "CustomEventType",
				Name:        "test-resource",
				Project:     "test-project",
				ObjectKind:  "CustomResource",
				ID:          "custom-event-id",
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventPrefix + "message":     "Custom event message",
				kargoapi.AnnotationKeyEventPrefix + "customField": "customValue",
				kargoapi.AnnotationKeyEventPrefix + "jsonField":   `{"key":"value"}`,
				kargoapi.AnnotationKeyEventPrefix + "type":        "CustomEventType",
				kargoapi.AnnotationKeyEventPrefix + "name":        "test-resource",
				kargoapi.AnnotationKeyEventPrefix + "project":     "test-project",
				kargoapi.AnnotationKeyEventPrefix + "objectKind":  "CustomResource",
				kargoapi.AnnotationKeyEventPrefix + "id":          "custom-event-id",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := convertToAnnotations(tc.data)

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

func TestConvertValueToString(t *testing.T) {
	testCases := map[string]struct {
		value    any
		expected string
	}{
		"string value": {
			value:    "test-string",
			expected: "test-string",
		},
		"map value": {
			value:    map[string]string{"key": "value"},
			expected: `{"key":"value"}`,
		},
		"slice value": {
			value:    []string{"item1", "item2"},
			expected: `["item1","item2"]`,
		},
		"number value": {
			value:    42,
			expected: "42",
		},
		"boolean value": {
			value:    true,
			expected: "true",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := convertValueToString(tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func isPromotionEvent(eventType kargoapi.EventType) bool {
	switch eventType {
	case kargoapi.EventTypePromotionCreated,
		kargoapi.EventTypePromotionSucceeded,
		kargoapi.EventTypePromotionFailed,
		kargoapi.EventTypePromotionErrored,
		kargoapi.EventTypePromotionAborted:
		return true
	default:
		return false
	}
}

func isVerificationEvent(eventType kargoapi.EventType) bool {
	switch eventType {
	case kargoapi.EventTypeFreightVerificationSucceeded,
		kargoapi.EventTypeFreightVerificationFailed,
		kargoapi.EventTypeFreightVerificationErrored,
		kargoapi.EventTypeFreightVerificationAborted,
		kargoapi.EventTypeFreightVerificationInconclusive,
		kargoapi.EventTypeFreightVerificationUnknown:
		return true
	default:
		return false
	}
}

// Mock EventRecorder for testing
type mockEventRecorder struct {
	callCount int
}

func (m *mockEventRecorder) Event(_ runtime.Object, _, _, _ string) {
	m.callCount++
}

func (m *mockEventRecorder) Eventf(_ runtime.Object, _, _, _ string, _ ...any) {
	m.callCount++
}

func (m *mockEventRecorder) AnnotatedEventf(_ runtime.Object, _ map[string]string,
	_, _, _ string, _ ...any,
) {
	m.callCount++
}

type customEvent struct {
	Message     string             `json:"message"`
	CustomField string             `json:"customField"`
	JSONField   map[string]string  `json:"jsonField"`
	EventType   kargoapi.EventType `json:"type"`
	Name        string             `json:"name"`
	Project     string             `json:"project"`
	ObjectKind  string             `json:"objectKind"`
	ID          string             `json:"id"`
}

func (c customEvent) Type() kargoapi.EventType {
	return c.EventType
}

func (c *customEvent) Kind() string {
	return c.ObjectKind
}

func (c *customEvent) GetName() string {
	return c.Name
}

func (c *customEvent) GetProject() string {
	return c.Project
}

func (c *customEvent) GetID() string {
	return c.ID
}
