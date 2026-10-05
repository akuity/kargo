package event

import (
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Common is a struct that contains fields common to all events.
type Common struct {
	Project string  `json:"project"`
	Actor   *string `json:"actor,omitempty"`
	Message string  `json:"message"`
}

func newCommonFromPromotion(message, actor string, promotion *kargoapi.Promotion) Common {
	if promotion == nil {
		return Common{}
	}
	evt := Common{
		Project: promotion.Namespace,
		Message: message,
	}
	if actor != "" {
		evt.Actor = &actor
	}
	// All Promotion-related events are emitted after the promotion was created.
	// Therefore, if the promotion knows who triggered it, set them as an actor.
	if promoteActor, ok := promotion.Annotations[kargoapi.AnnotationKeyCreateActor]; ok {
		evt.Actor = &promoteActor
	}
	return evt
}

func newCommonFromFreight(message,
	actor string, freight *kargoapi.Freight,
) Common {
	evt := Common{Message: message}
	if freight != nil {
		evt.Project = freight.GetNamespace()
	}
	if actor != "" {
		evt.Actor = &actor
	}
	return evt
}
