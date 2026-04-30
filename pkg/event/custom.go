package event

import (
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

const (
	// Well-known key values for custom events. If being pulled from Kubernetes annotations, it
	// should be prefixed with `AnnotationKeyEventPrefix` from the Kargo API

	AnnotationEventKeyKind    = "kind"
	AnnotationEventKeyName    = "name"
	AnnotationEventKeyProject = "project" // This is the same as the annotation, but just "project"
	AnnotationEventKeyMessage = "message"
)

// Custom is a representation of an arbitrary custom event. It contains the raw data from the
// message, deserialized as `map[string]any` as well as additional data that can be pulled from
// well-known keys. It implements the `Meta` and `Message` interfaces so it can be used by any of
// the other methods in the package.
//
// This type is only meant as a way to decode events. Sending custom events should be handled by
// defining your own structs
type Custom struct {
	ObjectKind string             `json:"kind"`
	Name       string             `json:"name"`
	Project    string             `json:"project"`
	EventType  kargoapi.EventType `json:"type"`
	Message    string             `json:"message,omitempty"`
	Data       map[string]any     `json:"data"`
	ID         string             `json:"id,omitempty"`
}

func (c *Custom) Kind() string {
	return c.ObjectKind
}

func (c *Custom) GetName() string {
	return c.Name
}

func (c *Custom) GetProject() string {
	return c.Project
}

func (c *Custom) Type() kargoapi.EventType {
	return c.EventType
}

func (c *Custom) GetMessage() string {
	return c.Message
}

func (c *Custom) SetMessage(msg string) {
	c.Message = msg
}

func (c *Custom) GetID() string {
	return c.ID
}
