package kubernetes

import (
	"context"
	"testing"
	"time"

	cloudevent "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
)

func TestFromKubernetesEvent(t *testing.T) {
	testCases := map[string]struct {
		k8sEvent     corev1.Event
		expectedType string
		expectedData any
		expectError  bool
		errorMessage string
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
			expectedType: event.EventTypePrefix + string(kargoapi.EventTypePromotionSucceeded),
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
			expectedType: event.EventTypePrefix + string(kargoapi.EventTypePromotionFailed),
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
			expectedType: event.EventTypePrefix + string(kargoapi.EventTypeFreightVerificationSucceeded),
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
			expectedType: event.EventTypePrefix + string(kargoapi.EventTypeFreightApproved),
		},
		"custom event type": {
			k8sEvent: corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					UID: "test-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyEventPrefix + "customField": "customValue",
						kargoapi.AnnotationKeyEventPrefix + "jsonField":   `{"key":"value"}`,
						"non-kargo-annotation":                            "ignored",
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
			expectedType: event.EventTypePrefix + "CustomEventType",
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
			require.Equal(t, string(tc.k8sEvent.UID), result.ID())
			require.Equal(t, tc.expectedType, result.Type())
			require.Equal(t, event.Source(tc.k8sEvent.InvolvedObject.Namespace,
				tc.k8sEvent.InvolvedObject.Kind, tc.k8sEvent.InvolvedObject.Name), result.Source())
			require.Equal(t, tc.k8sEvent.LastTimestamp.Time, result.Time())

			// Verify data is set correctly
			require.NotEmpty(t, result.Data())
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
			require.Equal(t, event.EventTypePrefix+string(eventType), result.Type())
			require.Equal(t, "test-uid", result.ID())
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
		cloudEvent    cloudevent.Event
		expectError   bool
		errorMessage  string
		expectedCalls int
	}{
		"promotion succeeded event": {
			cloudEvent: func() cloudevent.Event {
				evt := cloudevent.NewEvent()
				evt.SetType(event.EventTypePrefix + string(kargoapi.EventTypePromotionSucceeded))
				evt.SetSource(event.Source("test-project", "Promotion", "test-promotion"))

				promotionEvent := &event.PromotionSucceeded{
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
				}
				_ = evt.SetData(cloudevent.ApplicationJSON, promotionEvent)
				return evt
			}(),
			expectedCalls: 1,
		},
		"freight approved event": {
			cloudEvent: func() cloudevent.Event {
				evt := cloudevent.NewEvent()
				evt.SetType(event.EventTypePrefix + string(kargoapi.EventTypeFreightApproved))
				evt.SetSource(event.Source("test-project", "Freight", "test-freight"))

				freightEvent := &event.FreightApproved{
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
				}
				_ = evt.SetData(cloudevent.ApplicationJSON, freightEvent)
				return evt
			}(),
			expectedCalls: 1,
		},
		"custom event type": {
			cloudEvent: func() cloudevent.Event {
				evt := cloudevent.NewEvent()
				evt.SetType(event.EventTypePrefix + "CustomEventType")
				evt.SetSource(event.Source("test-project", "CustomResource", "test-resource"))

				data := map[string]any{
					"message":     "Custom event message",
					"customField": "customValue",
					"jsonField":   map[string]string{"key": "value"},
				}
				_ = evt.SetData(cloudevent.ApplicationJSON, data)
				return evt
			}(),
			expectedCalls: 1,
		},
		"invalid event type prefix": {
			cloudEvent: func() cloudevent.Event {
				evt := cloudevent.NewEvent()
				evt.SetType("invalid.prefix.event")
				evt.SetSource(event.Source("test-project", "Promotion", "test-promotion"))
				return evt
			}(),
			expectError:  true,
			errorMessage: "does not match expected prefix",
		},
		"invalid source format": {
			cloudEvent: func() cloudevent.Event {
				evt := cloudevent.NewEvent()
				evt.SetType(event.EventTypePrefix + string(kargoapi.EventTypePromotionSucceeded))
				evt.SetSource("invalid-source")

				promotionEvent := &event.PromotionSucceeded{
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
				}
				_ = evt.SetData(cloudevent.ApplicationJSON, promotionEvent)
				return evt
			}(),
			expectError:  true,
			errorMessage: "invalid event source",
		},
		"unsupported content type for custom event": {
			cloudEvent: func() cloudevent.Event {
				evt := cloudevent.NewEvent()
				evt.SetType(event.EventTypePrefix + "CustomEventType")
				evt.SetSource(event.Source("test-project", "CustomResource", "test-resource"))
				_ = evt.SetData("text/plain", "plain text data")
				return evt
			}(),
			expectError:  true,
			errorMessage: "unsupported content type",
		},
		"invalid event data": {
			cloudEvent: func() cloudevent.Event {
				evt := cloudevent.NewEvent()
				evt.SetType(event.EventTypePrefix + string(kargoapi.EventTypePromotionSucceeded))
				evt.SetSource(event.Source("test-project", "Promotion", "test-promotion"))
				_ = evt.SetData(cloudevent.ApplicationJSON, "invalid-data")
				return evt
			}(),
			expectError:  true,
			errorMessage: "failed to unmarshal event data",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			recorder := &mockEventRecorder{}
			sender := NewEventSender(recorder)

			err := sender.Send(context.Background(), tc.cloudEvent)

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
		data         []byte
		expected     map[string]string
		expectError  bool
		errorMessage string
	}{
		"simple data": {
			data: []byte(`{"field1":"value1","field2":"value2"}`),
			expected: map[string]string{
				kargoapi.AnnotationKeyEventPrefix + "field1": "value1",
				kargoapi.AnnotationKeyEventPrefix + "field2": "value2",
			},
		},
		"complex data": {
			data: []byte(`{"stringField":"value","objectField":{"key":"value"},"arrayField":["item1","item2"]}`),
			expected: map[string]string{
				kargoapi.AnnotationKeyEventPrefix + "stringField": "value",
				kargoapi.AnnotationKeyEventPrefix + "objectField": `{"key":"value"}`,
				kargoapi.AnnotationKeyEventPrefix + "arrayField":  `["item1","item2"]`,
			},
		},
		"invalid json": {
			data:         []byte(`invalid json`),
			expectError:  true,
			errorMessage: "failed to unmarshal event data",
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
