package kubernetes

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	cloudevent "github.com/cloudevents/sdk-go/v2"
	corev1 "k8s.io/api/core/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
)

// FromKubernetesEvent converts a Kubernetes Kargo event to a CloudEvent.
func FromKubernetesEvent(evt corev1.Event) (cloudevent.Event, error) {
	cloudEvent := cloudevent.NewEvent()
	cloudEvent.SetID(string(evt.UID))
	cloudEvent.SetSource(event.Source(evt.InvolvedObject.Namespace, evt.InvolvedObject.Kind, evt.InvolvedObject.Name))
	cloudEvent.SetType(event.EventTypePrefix + evt.Reason)
	cloudEvent.SetTime(evt.LastTimestamp.Time)

	if slices.Contains(event.KnownPromotionEventTypes, kargoapi.EventType(evt.Reason)) {
		// If the event reason is a built-in event type, we can use the PromotionEvent struct
		data, err := event.UnmarshalPromotionEventAnnotations(evt.Annotations)
		if err != nil {
			return cloudevent.Event{}, fmt.Errorf("failed to unmarshal event annotations: %w", err)
		}
		data.Message = evt.Message
		if err := cloudEvent.SetData(cloudevent.ApplicationJSON, data); err != nil {
			return cloudevent.Event{}, fmt.Errorf("failed to set event data: %w", err)
		}
	} else if slices.Contains(event.KnownFreightEventTypes, kargoapi.EventType(evt.Reason)) {
		// If the event reason is a built-in freight event type, we can use the FreightEvent struct
		data, err := event.UnmarshalFreightEventAnnotations(evt.Annotations)
		if err != nil {
			return cloudevent.Event{}, fmt.Errorf("failed to unmarshal event annotations: %w", err)
		}
		data.Message = evt.Message
		if err := cloudEvent.SetData(cloudevent.ApplicationJSON, data); err != nil {
			return cloudevent.Event{}, fmt.Errorf("failed to set event data: %w", err)
		}
	} else {
		// For custom event types, we try to parse everything back out to a generic
		// map[string]interface{} so it can be parsed into a concrete type by a consumer
		data := make(map[string]any)
		for k, v := range evt.Annotations {
			if !strings.HasPrefix(k, kargoapi.AnnotationKeyEventPrefix) {
				// Skip non-Kargo annotations
				continue
			}
			// Always try to JSON unmarshal the value, if it fails, we just store the string. Pretty
			// much any other type will unmarshal properly other than a string
			var val any
			if err := json.Unmarshal([]byte(v), &val); err != nil {
				// Unmarshaling into an interface should always succeed, so any error here is likely
				// due to the value not being valid JSON
				val = v
			}
			// Strip the annotation prefix
			data[strings.TrimPrefix(k, kargoapi.AnnotationKeyEventPrefix)] = val
		}
		if err := cloudEvent.SetData(cloudevent.ApplicationJSON, data); err != nil {
			return cloudevent.Event{}, fmt.Errorf("failed to set event data: %w", err)
		}
	}

	return cloudEvent, nil
}
