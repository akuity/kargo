package v1alpha1

import (
	"context"
	"encoding/json"
	"time"

	authnv1 "k8s.io/api/authentication/v1"

	"github.com/akuity/kargo/internal/api/user"
	"github.com/akuity/kargo/internal/logging"
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

// NewPromotionEventAnnotations returns annotations for a Promotion related event.
// It may skip some fields when error occurred during serialization, to record event with best-effort.
func NewPromotionEventAnnotations(
	ctx context.Context,
	actor string,
	p *Promotion,
	f *Freight,
) map[string]string {
	log := logging.LoggerFromContext(ctx)

	annotations := map[string]string{
		AnnotationKeyEventProject:             p.GetNamespace(),
		AnnotationKeyEventPromotionName:       p.GetName(),
		AnnotationKeyEventFreightName:         p.Spec.Freight,
		AnnotationKeyEventStageName:           p.Spec.Stage,
		AnnotationKeyEventPromotionCreateTime: p.GetCreationTimestamp().Format(time.RFC3339),
	}
	if actor != "" {
		annotations[AnnotationKeyEventActor] = actor
	}
	if f != nil {
		annotations[AnnotationKeyEventFreightCreateTime] = f.CreationTimestamp.Format(time.RFC3339)
		annotations[AnnotationKeyEventFreightAlias] = f.Alias
		if len(f.Commits) > 0 {
			data, err := json.Marshal(f.Commits)
			if err != nil {
				log.WithError(err).Error("marshal freight commits in JSON")
			} else {
				annotations[AnnotationKeyEventFreightCommits] = string(data)
			}
		}
		if len(f.Images) > 0 {
			data, err := json.Marshal(f.Images)
			if err != nil {
				log.WithError(err).Error("marshal freight images in JSON")
			} else {
				annotations[AnnotationKeyEventFreightImages] = string(data)
			}
		}
		if len(f.Charts) > 0 {
			data, err := json.Marshal(f.Charts)
			if err != nil {
				log.WithError(err).Error("marshal freight charts in JSON")
			} else {
				annotations[AnnotationKeyEventFreightCharts] = string(data)
			}
		}
	}
	return annotations
}
