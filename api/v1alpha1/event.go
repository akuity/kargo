package v1alpha1

import (
	authnv1 "k8s.io/api/authentication/v1"

	"github.com/akuity/kargo/internal/api/user"
)

const (
	EventReasonPromotionCreated                = "PromotionCreated"
	EventReasonPromotionSucceeded              = "PromotionSucceeded"
	EventReasonPromotionFailed                 = "PromotionFailed"
	EventReasonPromotionErrored                = "PromotionErrored"
	EventReasonFreightApproved                 = "FreightApproved"
	EventReasonFreightVerificationSucceeded    = "FreightVerificationSucceeded"
	EventReasonFreightVerificationFailed       = "FreightVerificationFailed"
	EventReasonFreightVerificationErrored      = "FreightVerificationErrored"
	EventReasonFreightVerificationAborted      = "FreightVerificationAborted"
	EventReasonFreightVerificationInconclusive = "FreightVerificationInconclusive"
	EventReasonFreightVerificationUnknown      = "FreightVerificationUnknown"
)

const (
	EventActorAdmin                = "admin"
	EventActorControllerPrefix     = "controller:"
	EventActorEmailPrefix          = "email:"
	EventActorSubjectPrefix        = "subject:"
	EventActorKubernetesUserPrefix = "kubernetes:"
	EventActorUnknown              = "unknown actor"
)

func FormatEventControllerActor(name string) string {
	return EventActorControllerPrefix + name
}

// FormatEventUserActor returns a string representation of the user acting in an event
// that can be used as a value of AnnotationKeyEventActor.
//
// 1. If the user is admin, it returns EventActorAdmin since other information is not available.
// 2. If the email is available, it returns email in "email:<email>" format.
// 3. If the subject is available, it returns subject in "subject:<subject>" format.
// 4. Otherwise, it returns EventActorUnknown.
func FormatEventUserActor(u user.Info) string {
	switch {
	case u.IsAdmin:
		return EventActorAdmin
	case u.Email != "":
		return EventActorEmailPrefix + u.Email
	case u.Subject != "":
		return EventActorSubjectPrefix + u.Subject
	default:
		return EventActorUnknown
	}
}

func FormatEventKubernetesUserActor(u authnv1.UserInfo) string {
	return EventActorKubernetesUserPrefix + u.Username
}

func NewFreightApprovedEventAnnotations(actor string, f *Freight, stageName string) map[string]string {
	annotations := map[string]string{
		AnnotationKeyEventProject:      f.Namespace,
		AnnotationKeyEventFreightAlias: f.Alias,
		AnnotationKeyEventFreightName:  f.Name,
		AnnotationKeyEventStageName:    stageName,
	}
	if actor != "" {
		annotations[AnnotationKeyEventActor] = actor
	}
	return annotations
}

func NewPromotionCreatedEventAnnotations(actor string, p *Promotion, f *Freight) map[string]string {
	annotations := map[string]string{
		AnnotationKeyEventActor:         actor,
		AnnotationKeyEventProject:       p.Namespace,
		AnnotationKeyEventPromotionName: p.Name,
		AnnotationKeyEventFreightName:   p.Spec.Freight,
		AnnotationKeyEventStageName:     p.Spec.Stage,
	}
	if actor != "" {
		annotations[AnnotationKeyEventActor] = actor
	}
	if f != nil {
		annotations[AnnotationKeyEventFreightAlias] = f.Alias
	}
	return annotations
}
