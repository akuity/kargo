package kubernetes

import (
	"encoding/json"
	"strings"

	corev1 "k8s.io/api/core/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
)

// FromKubernetesEvent converts a Kubernetes Kargo event to a concrete event type
func FromKubernetesEvent(evt corev1.Event) (event.Meta, error) {
	var parsedEvent event.Meta
	var err error

	id := string(evt.UID)
	switch kargoapi.EventType(evt.Reason) {
	case kargoapi.EventTypePromotionCreated:
		parsedEvent, err = event.UnmarshalPromotionCreatedAnnotations(id, evt.Annotations)
	case kargoapi.EventTypePromotionSucceeded:
		parsedEvent, err = event.UnmarshalPromotionSucceededAnnotations(id, evt.Annotations)
	case kargoapi.EventTypePromotionFailed:
		parsedEvent, err = event.UnmarshalPromotionFailedAnnotations(id, evt.Annotations)
	case kargoapi.EventTypePromotionErrored:
		parsedEvent, err = event.UnmarshalPromotionErroredAnnotations(id, evt.Annotations)
	case kargoapi.EventTypePromotionAborted:
		parsedEvent, err = event.UnmarshalPromotionAbortedAnnotations(id, evt.Annotations)
	case kargoapi.EventTypeFreightApproved:
		parsedEvent, err = event.UnmarshalFreightApprovedAnnotations(id, evt.Annotations)
	case kargoapi.EventTypeFreightVerificationSucceeded:
		parsedEvent, err = event.UnmarshalFreightVerificationSucceededAnnotations(id, evt.Annotations)
	case kargoapi.EventTypeFreightVerificationFailed:
		parsedEvent, err = event.UnmarshalFreightVerificationFailedAnnotations(id, evt.Annotations)
	case kargoapi.EventTypeFreightVerificationErrored:
		parsedEvent, err = event.UnmarshalFreightVerificationErroredAnnotations(id, evt.Annotations)
	case kargoapi.EventTypeFreightVerificationAborted:
		parsedEvent, err = event.UnmarshalFreightVerificationAbortedAnnotations(id, evt.Annotations)
	case kargoapi.EventTypeFreightVerificationInconclusive:
		parsedEvent, err = event.UnmarshalFreightVerificationInconclusiveAnnotations(id, evt.Annotations)
	case kargoapi.EventTypeFreightVerificationUnknown:
		parsedEvent, err = event.UnmarshalFreightVerificationUnknownAnnotations(id, evt.Annotations)
	default:
		customEvt := &event.Custom{
			EventType: kargoapi.EventType(evt.Reason),
			ID:        id,
		}
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
			if unmarshalErr := json.Unmarshal([]byte(v), &val); unmarshalErr != nil {
				// Unmarshaling into an interface should always succeed, so any error here is likely
				// due to the value not being valid JSON
				val = v
			}
			// Strip the annotation prefix
			key := strings.TrimPrefix(k, kargoapi.AnnotationKeyEventPrefix)
			// If the value is a string, check for a well-known annotation
			if valStr, ok := val.(string); ok {
				switch key {
				case event.AnnotationEventKeyKind:
					customEvt.ObjectKind = valStr
				case event.AnnotationEventKeyName:
					customEvt.Name = valStr
				case event.AnnotationEventKeyProject:
					customEvt.Project = valStr
				case event.AnnotationEventKeyMessage:
					customEvt.Message = valStr
				}
				continue
			}
			data[key] = val
		}
		customEvt.Data = data
		parsedEvent = customEvt
	}

	if err != nil {
		return nil, err
	}

	return parsedEvent, nil
}
