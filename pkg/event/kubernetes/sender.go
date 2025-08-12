package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	cloudevent "github.com/cloudevents/sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
)

type EventSender struct {
	recorder record.EventRecorder
}

// NewEventSender creates a new EventSender that uses the provided EventRecorder
func NewEventSender(recorder record.EventRecorder) *EventSender {
	return &EventSender{
		recorder: recorder,
	}
}

// Send sends the event to the Kubernetes EventRecorder
func (s *EventSender) Send(_ context.Context, evt cloudevent.Event) error {
	eventType := evt.Type()
	if !strings.HasPrefix(eventType, event.EventTypePrefix) {
		return fmt.Errorf("event type %s does not match expected prefix %s", eventType, event.EventTypePrefix)
	}
	eventType = strings.TrimPrefix(eventType, event.EventTypePrefix)
	// First up, extract event data. If it is our known event type, we can deserialize, if it
	// doesn't match, we do a generic conversion to annotations
	var annotations map[string]string
	var message string
	if slices.Contains(event.KnownPromotionEventTypes, kargoapi.EventType(eventType)) {
		var data event.PromotionEvent
		if err := evt.DataAs(&data); err != nil {
			return fmt.Errorf("failed to unmarshal event data: %w", err)
		}
		message = data.Message
		annotations = data.MarshalAnnotations()
	} else if slices.Contains(event.KnownFreightEventTypes, kargoapi.EventType(eventType)) {
		var data event.FreightEvent
		if err := evt.DataAs(&data); err != nil {
			return fmt.Errorf("failed to unmarshal event data: %w", err)
		}
		message = data.Message
		annotations = data.MarshalAnnotations()
	} else {
		if evt.DataContentType() != cloudevent.ApplicationJSON {
			return fmt.Errorf("unsupported content type %s for event type %s", evt.DataContentType(), eventType)
		}
		var err error
		annotations, err = convertToAnnotations(evt.Data())
		if err != nil {
			return fmt.Errorf("failed to convert event data to annotations: %w", err)
		}
		// Check for a message key in the annotations, if it exists, we use it as the message
		// and remove it from the annotations
		if msg, ok := annotations[kargoapi.AnnotationKeyEventPrefix+"message"]; ok {
			message = msg
			delete(annotations, kargoapi.AnnotationKeyEventPrefix+"message")
		}
	}

	// Now that we have the data and the type, put together an object reference for the event
	parts := strings.Split(evt.Source(), "/")
	if len(parts) != 3 {
		return fmt.Errorf("invalid event source: %s", evt.Source())
	}
	reference := corev1.ObjectReference{
		Namespace: parts[0],
		Kind:      parts[1],
		Name:      parts[2],
	}
	// Use the recorder to send the event
	s.recorder.AnnotatedEventf(&reference, annotations, corev1.EventTypeNormal, eventType, message)

	return nil
}

// convertToAnnotations takes raw JSON data and decodes it to a `map[string]interface{}`. It then
// converts raw types to string and JSON encodes anything else that isn't a raw type (e.g. maps,
// slices, structs). It will take each data key and prefix it with
// `kargoapi.AnnotationEventKeyPrefix` and then return the map.
func convertToAnnotations(data []byte) (map[string]string, error) {
	var rawData map[string]any
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	annotations := make(map[string]string)
	for key, value := range rawData {
		annotationKey := kargoapi.AnnotationKeyEventPrefix + key
		annotationValue, err := convertValueToString(value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value for key %s: %w", key, err)
		}
		annotations[annotationKey] = annotationValue
	}

	return annotations, nil
}

// convertValueToString converts a value to a string representation. If the value is a string, it
// returns it directly. Otherwise, it JSON encodes the value.
func convertValueToString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	default:
		// For complex types (maps, slices, structs), JSON encode them
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal value to JSON: %w", err)
		}
		return string(jsonBytes), nil
	}
}
