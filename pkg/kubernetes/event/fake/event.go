package fake

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

type Event struct {
	Annotations map[string]string
	EventType   string
	Reason      string
	Message     string
	Object      runtime.Object
}

func NewEvent(
	annotations map[string]string,
	eventType,
	reason,
	message string,
	object runtime.Object,
) Event {
	return Event{
		Annotations: annotations,
		EventType:   eventType,
		Reason:      reason,
		Message:     message,
		Object:      object,
	}
}

var (
	_ record.EventRecorder = &EventRecorder{}
)

type EventRecorder struct {
	Events chan Event
}

func NewEventRecorder(bufferSize int) *EventRecorder {
	return &EventRecorder{
		Events: make(chan Event, bufferSize),
	}
}

func (r *EventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	if r.Events != nil {
		r.Events <- NewEvent(nil, eventtype, reason, message, object)
	}
}

func (r *EventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...any) {
	if r.Events != nil {
		r.Events <- NewEvent(nil, eventtype, reason, fmt.Sprintf(messageFmt, args...), object)
	}
}

func (r *EventRecorder) AnnotatedEventf(
	object runtime.Object,
	annotations map[string]string,
	eventtype,
	reason,
	messageFmt string,
	args ...any,
) {
	if r.Events != nil {
		r.Events <- NewEvent(annotations, eventtype, reason, fmt.Sprintf(messageFmt, args...), object)
	}
}
