package v1alpha1

import (
	"github.com/akuity/kargo/internal/api/user"
)

const (
	EventReasonPromotionCreated       = "PromotionCreated"
	EventReasonFreightApproved        = "FreightApproved"
	EventReasonFreightVerifiedInStage = "FreightVerifiedInStage"
)

const (
	EventActorAdmin         = "admin"
	EventActorEmailPrefix   = "email:"
	EventActorSubjectPrefix = "subject:"
	EventActorUnknown       = "unknown actor"
)

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
