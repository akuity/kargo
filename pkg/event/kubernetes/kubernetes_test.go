package kubernetes

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
)

func TestFromKubernetesEvent(t *testing.T) {
	// We're going to be sneaky here and use a normal PromotionEvent, but then label it as a custom
	// event type so it gets treated as if it were a custom event type.

	testPromotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-promotion",
			Namespace: "test-namespace",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			},
		},
		Spec: kargoapi.PromotionSpec{
			Freight: "test-freight",
			Stage:   "test-stage",
		},
	}

	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-freight",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			},
		},
		Alias: "test-alias",
	}

	promotionEvent := event.NewPromotionEvent("test message", "test-actor", testPromotion, testFreight)
	cloudEvent := promotionEvent.ToCloudEvent(kargoapi.EventTypePromotionSucceeded)
	fakeRecorder := record.NewFakeRecorder(10)
	sender := NewEventSender(fakeRecorder)

	err := sender.Send(context.Background(), cloudEvent)
	require.NoError(t, err)

	select {
	case <-fakeRecorder.Events:
		// If we reach here, it means an event was recorded, now we have to generate a fake event
		// that mimics a Kubernetes event with a custom event type.
		kubernetesEvent := corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				UID:         "test-uid",
				Annotations: promotionEvent.MarshalAnnotations(),
			},
			InvolvedObject: corev1.ObjectReference{
				Namespace: "test-namespace",
				Kind:      "Promotion",
				Name:      "test-promotion",
			},
			Reason:        "CustomEventType",
			Message:       "test message",
			LastTimestamp: metav1.Time{Time: cloudEvent.Time()},
		}

		convertedEvent, err := FromKubernetesEvent(kubernetesEvent)
		require.NoError(t, err)

		// Assert that the event type is correct and the data matches the original PromotionEvent
		expectedEventType := event.EventTypePrefix + "CustomEventType"
		require.Equal(t, expectedEventType, convertedEvent.Type())

		// Since this is a custom event type, the data should be a map[string]interface{}
		var eventData map[string]any
		err = convertedEvent.DataAs(&eventData)
		require.NoError(t, err)

		// Verify some key fields from the original promotion event are present
		require.Equal(t, "test-namespace", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventProject, kargoapi.AnnotationKeyEventPrefix)])
		require.Equal(t, "test-promotion", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventPromotionName, kargoapi.AnnotationKeyEventPrefix)])
		require.Equal(t, "test-freight", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventFreightName, kargoapi.AnnotationKeyEventPrefix)])
		require.Equal(t, "test-stage", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventStageName, kargoapi.AnnotationKeyEventPrefix)])
		require.Equal(t, "test-actor", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventActor, kargoapi.AnnotationKeyEventPrefix)])

		// Assert that the event source is correct
		expectedSource := event.Source("test-namespace", "Promotion", "test-promotion")
		require.Equal(t, expectedSource, convertedEvent.Source())

		// Verify the event ID matches the Kubernetes event UID
		require.Equal(t, "test-uid", convertedEvent.ID())

	default:
		t.Fatal("Expected an event to be recorded")
	}
}

func TestPromotionEventRoundTrip(t *testing.T) {
	testPromotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-promotion",
			Namespace: "test-namespace",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			},
		},
		Spec: kargoapi.PromotionSpec{
			Freight: "test-freight",
			Stage:   "test-stage",
		},
	}

	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-freight",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			},
		},
		Alias: "test-alias",
	}

	promotionEvent := event.NewPromotionEvent("test message", "test-actor", testPromotion, testFreight)
	kubernetesEvent := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			UID:         "test-uid",
			Annotations: promotionEvent.MarshalAnnotations(),
		},
		InvolvedObject: corev1.ObjectReference{
			Namespace: "test-namespace",
			Kind:      "Promotion",
			Name:      "test-promotion",
		},
		Reason:        string(event.KnownPromotionEventTypes[0]),
		Message:       "test message",
		LastTimestamp: metav1.Time{Time: time.Now()},
	}

	convertedEvent, err := FromKubernetesEvent(kubernetesEvent)
	require.NoError(t, err)

	// Now assert that the converted event matches the original PromotionEvent
	var roundTripped event.PromotionEvent
	err = convertedEvent.DataAs(&roundTripped)
	require.NoError(t, err)
	require.Equal(t, promotionEvent, roundTripped)
}

func TestFreightEventRoundTrip(t *testing.T) {
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-freight",
			Namespace: "test-namespace",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			},
		},
		Alias:   "v1.2.3",
		Commits: []kargoapi.GitCommit{{Tag: "v1.2.3", ID: "abc123"}},
		Images:  []kargoapi.Image{{Tag: "v1.2.3", RepoURL: "example.com/app"}},
		Charts:  []kargoapi.Chart{{Name: "my-chart", Version: "1.2.3"}},
	}

	freightEvent := event.NewFreightEvent("test-actor", testFreight, "test-stage", "test message")

	// Add verification timing
	startTime := time.Date(2024, 10, 22, 1, 0, 0, 0, time.UTC)
	finishTime := time.Date(2024, 10, 22, 2, 0, 0, 0, time.UTC)
	freightEvent.VerificationStartTime = &startTime
	freightEvent.VerificationFinishTime = &finishTime

	kubernetesEvent := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			UID:         "test-freight-uid",
			Annotations: freightEvent.MarshalAnnotations(),
		},
		InvolvedObject: corev1.ObjectReference{
			Namespace: "test-namespace",
			Kind:      "Freight",
			Name:      "test-freight",
		},
		Reason:        string(event.KnownFreightEventTypes[0]), // FreightApproved
		Message:       "test message",
		LastTimestamp: metav1.Time{Time: time.Now()},
	}

	convertedEvent, err := FromKubernetesEvent(kubernetesEvent)
	require.NoError(t, err)

	// Now assert that the converted event matches the original FreightEvent
	var roundTripped event.FreightEvent
	err = convertedEvent.DataAs(&roundTripped)
	require.NoError(t, err)
	require.Equal(t, freightEvent, roundTripped)
}

func TestFromKubernetesEventWithFreight(t *testing.T) {
	// Create a test freight event similar to the promotion test but for freight
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-freight",
			Namespace: "test-namespace",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			},
		},
		Alias:   "v1.2.3",
		Commits: []kargoapi.GitCommit{{Tag: "v1.2.3", ID: "abc123"}},
		Images:  []kargoapi.Image{{Tag: "v1.2.3", RepoURL: "example.com/app"}},
		Charts:  []kargoapi.Chart{{Name: "my-chart", Version: "1.2.3"}},
	}

	freightEvent := event.NewFreightEvent("test-actor", testFreight, "test-stage", "test message")
	cloudEvent := freightEvent.ToCloudEvent(kargoapi.EventTypeFreightApproved)
	fakeRecorder := record.NewFakeRecorder(10)
	sender := NewEventSender(fakeRecorder)

	err := sender.Send(context.Background(), cloudEvent)
	require.NoError(t, err)

	select {
	case <-fakeRecorder.Events:
		// If we reach here, it means an event was recorded, now we have to generate a fake event
		// that mimics a Kubernetes event with a custom freight event type.
		kubernetesEvent := corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				UID:         "test-freight-uid",
				Annotations: freightEvent.MarshalAnnotations(),
			},
			InvolvedObject: corev1.ObjectReference{
				Namespace: "test-namespace",
				Kind:      "Freight",
				Name:      "test-freight",
			},
			Reason:        "CustomFreightEventType",
			Message:       "test message",
			LastTimestamp: metav1.Time{Time: cloudEvent.Time()},
		}

		convertedEvent, err := FromKubernetesEvent(kubernetesEvent)
		require.NoError(t, err)

		// Assert that the event type is correct and the data matches the original FreightEvent
		expectedEventType := event.EventTypePrefix + "CustomFreightEventType"
		require.Equal(t, expectedEventType, convertedEvent.Type())

		// Since this is a custom event type, the data should be a map[string]interface{}
		var eventData map[string]any
		err = convertedEvent.DataAs(&eventData)
		require.NoError(t, err)

		// Verify some key fields from the original freight event are present
		require.Equal(t, "test-namespace", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventProject, kargoapi.AnnotationKeyEventPrefix)])
		require.Equal(t, "test-freight", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventFreightName, kargoapi.AnnotationKeyEventPrefix)])
		require.Equal(t, "test-stage", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventStageName, kargoapi.AnnotationKeyEventPrefix)])
		require.Equal(t, "test-actor", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventActor, kargoapi.AnnotationKeyEventPrefix)])
		require.Equal(t, "v1.2.3", eventData[strings.TrimPrefix(
			kargoapi.AnnotationKeyEventFreightAlias, kargoapi.AnnotationKeyEventPrefix)])

		// Assert that the event source is correct
		expectedSource := event.Source("test-namespace", "Freight", "test-freight")
		require.Equal(t, expectedSource, convertedEvent.Source())

		// Verify the event ID matches the Kubernetes event UID
		require.Equal(t, "test-freight-uid", convertedEvent.ID())

	default:
		t.Fatal("Expected an event to be recorded")
	}
}
