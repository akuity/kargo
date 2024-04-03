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
	EventActorSubjectPrefix = "email:"
	EventActorUnknown       = "unknown actor"
)

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
