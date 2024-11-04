package v1alpha1

import (
	"time"

	authnv1 "k8s.io/api/authentication/v1"

	"github.com/akuity/kargo/internal/api/user"
)

const (
	AnnotationKeyEventActor                  = "event.kargo.akuity.io/actor"
	AnnotationKeyEventProject                = "event.kargo.akuity.io/project"
	AnnotationKeyEventPromotionName          = "event.kargo.akuity.io/promotion-name"
	AnnotationKeyEventPromotionCreateTime    = "event.kargo.akuity.io/promotion-create-time"
	AnnotationKeyEventFreightAlias           = "event.kargo.akuity.io/freight-alias"
	AnnotationKeyEventFreightName            = "event.kargo.akuity.io/freight-name"
	AnnotationKeyEventFreightCreateTime      = "event.kargo.akuity.io/freight-create-time"
	AnnotationKeyEventFreightCommits         = "event.kargo.akuity.io/freight-commits"
	AnnotationKeyEventFreightImages          = "event.kargo.akuity.io/freight-images"
	AnnotationKeyEventFreightCharts          = "event.kargo.akuity.io/freight-charts"
	AnnotationKeyEventStageName              = "event.kargo.akuity.io/stage-name"
	AnnotationKeyEventAnalysisRunName        = "event.kargo.akuity.io/analysis-run-name"
	AnnotationKeyEventVerificationPending    = "event.kargo.akuity.io/verification-pending"
	AnnotationKeyEventVerificationStartTime  = "event.kargo.akuity.io/verification-start-time"
	AnnotationKeyEventVerificationFinishTime = "event.kargo.akuity.io/verification-finish-time"
	AnnotationKeyEventApplications           = "event.kargo.akuity.io/applications"
)

const (
	EventReasonPromotionCreated                = "PromotionCreated"
	EventReasonPromotionSucceeded              = "PromotionSucceeded"
	EventReasonPromotionFailed                 = "PromotionFailed"
	EventReasonPromotionErrored                = "PromotionErrored"
	EventReasonPromotionAborted                = "PromotionAborted"
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
	var email, subject string
	if emailClaim, ok := u.Claims["email"]; ok {
		if emailStr, ok := emailClaim.(string); ok {
			email = emailStr
		}
	}
	if subClaim, ok := u.Claims["sub"]; ok {
		if subStr, ok := subClaim.(string); ok {
			subject = subStr
		}
	}

	switch {
	case u.IsAdmin:
		return EventActorAdmin
	case email != "":
		return EventActorEmailPrefix + email
	case subject != "":
		return EventActorSubjectPrefix + subject
	default:
		return EventActorUnknown
	}
}

func FormatEventKubernetesUserActor(u authnv1.UserInfo) string {
	return EventActorKubernetesUserPrefix + u.Username
}

func NewFreightApprovedEventAnnotations(actor string, f *Freight, stageName string) map[string]string {
	annotations := map[string]string{
		AnnotationKeyEventProject:           f.Namespace,
		AnnotationKeyEventFreightCreateTime: f.CreationTimestamp.Format(time.RFC3339),
		AnnotationKeyEventFreightAlias:      f.Alias,
		AnnotationKeyEventFreightName:       f.Name,
		AnnotationKeyEventStageName:         stageName,
	}
	if actor != "" {
		annotations[AnnotationKeyEventActor] = actor
	}
	return annotations
}
