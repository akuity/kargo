package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/fatih/structtag"
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
func (s *EventSender) Send(_ context.Context, evt event.Meta) error {
	// First up, extract event data. If it is our known event type, we can deserialize, if it
	// doesn't match, we do a generic conversion to annotations
	var annotations map[string]string
	var message string
	var err error

	var apiVersion string

	// NOTE(thomastaylor312): My kingdom for an enum here so we would know if we exhausted all
	// cases. Whenever we add a new type, it should be added here or it will fall into default
	// handling
	if slices.Contains(event.KnownEventTypes, evt.Type()) {
		// Type assert that the event is the type we expect for Kubernetes. If it is a known event
		// type, it should implement this interface
		typedEvt, ok := evt.(unionedInterface)
		if !ok {
			return fmt.Errorf("failed to assert event type, this is programmer error: %T", evt)
		}
		annotations, message = toAnnotationsAndMessage(typedEvt)
		// We add the apiVersion here if it is a known type as we know it is coming from a Kargo event
		apiVersion = kargoapi.GroupVersion.Identifier()
	} else {
		annotations, err = convertToAnnotations(evt)
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
	if err != nil {
		return fmt.Errorf("failed to extract event data: %w", err)
	}
	reference := corev1.ObjectReference{
		Namespace:  evt.GetProject(),
		Kind:       evt.Kind(),
		Name:       evt.GetName(),
		APIVersion: apiVersion,
	}
	// Use the recorder to send the event
	s.recorder.AnnotatedEventf(&reference, annotations, corev1.EventTypeNormal, string(evt.Type()), message)

	return nil
}

type unionedInterface interface {
	event.AnnotationMarshaler
	event.Message
	event.Meta
}

func toAnnotationsAndMessage(evt unionedInterface) (map[string]string, string) {
	message := evt.GetMessage()
	annotations := evt.MarshalAnnotations()

	return annotations, message
}

// convertToAnnotations uses reflection to iterate over the fields of the given struct and converts
// them to annotations. It skips any fields that have nil values. It will take each field name and
// prefix it with `kargoapi.AnnotationEventKeyPrefix` and then return the map.
func convertToAnnotations(data event.Meta) (map[string]string, error) {

	// NOTE(thomastaylor312): I tried to do this
	annotations := make(map[string]string)

	rv := reflect.ValueOf(data)
	rt := reflect.TypeOf(data)

	// Handle pointer to struct
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return annotations, nil
		}
		rv = rv.Elem()
		rt = rt.Elem()
	}

	// Ensure we're working with a struct
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", rv.Kind())
	}

	// Iterate over struct fields
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// Shouldn't happen, but for safety sake we check
		if !field.IsValid() {
			continue
		}

		// Skip nil pointer fields
		if field.Kind() == reflect.Pointer && field.IsNil() {
			continue
		}

		// Skip nil interface fields
		if field.Kind() == reflect.Interface && field.IsNil() {
			continue
		}

		val := field.Interface()

		// Unalias any string types for this next step so they are processed properly
		if field.Kind() == reflect.String {
			val = field.String()
		}

		// Get the field name from the JSON tag or the struct field name
		fieldName := getFieldName(fieldType)

		// Convert field value to string
		annotationKey := kargoapi.AnnotationKeyEventPrefix + fieldName
		annotationValue, err := convertValueToString(val)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value for field %s: %w", fieldName, err)
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
	case *string:
		if v != nil {
			return *v, nil
		}
		return "", nil
	// Special cases: the Event
	default:
		// For complex types (maps, slices, structs), JSON encode them
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal value to JSON: %w", err)
		}
		return string(jsonBytes), nil
	}
}

// Get field name from the JSON tag or the struct field name
func getFieldName(field reflect.StructField) string {
	tags, err := structtag.Parse(string(field.Tag))
	if err != nil {
		return field.Name
	}
	jsonTag, err := tags.Get("json")
	if err != nil {
		return field.Name
	}
	return jsonTag.Name
}
