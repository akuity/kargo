package event

import (
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// The reverse DNS event type prefix used for all Kargo CloudEvents.
const EventTypePrefix = "io.akuity.kargo.event."

// KnownPromotionEventTypes contains the list of built-in event types that are recognized by the system.
// This is used for marshal/unmarshal operations of various events
var KnownPromotionEventTypes = []kargoapi.EventType{
	kargoapi.EventTypePromotionCreated,
	kargoapi.EventTypePromotionSucceeded,
	kargoapi.EventTypePromotionFailed,
	kargoapi.EventTypePromotionErrored,
	kargoapi.EventTypePromotionAborted,
}

// KnownFreightEventTypes contains the list of built-in freight event types that are recognized by the system.
var KnownFreightEventTypes = []kargoapi.EventType{
	kargoapi.EventTypeFreightApproved,
	kargoapi.EventTypeFreightVerificationSucceeded,
	kargoapi.EventTypeFreightVerificationFailed,
	kargoapi.EventTypeFreightVerificationErrored,
	kargoapi.EventTypeFreightVerificationAborted,
	kargoapi.EventTypeFreightVerificationInconclusive,
	kargoapi.EventTypeFreightVerificationUnknown,
}

// Source is a utility function that formats the source of an event (from a Kubernetes object). It combines
// the namespace, kind, and name of the event into a single string for use in a CloudEvent
func Source(namespace, kind, name string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, kind, name)
}
