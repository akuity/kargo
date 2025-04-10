package api

import (
	"time"

	authnv1 "k8s.io/api/authentication/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/user"
)

func FormatEventControllerActor(name string) string {
	return kargoapi.EventActorControllerPrefix + name
}

// FormatEventUserActor returns a string representation of the user acting in an event
// that can be used as a value of AnnotationKeyEventActor.
//
// 1. If the user is admin, it returns EventActorAdmin since other information is not available.
// 2. If the email is available, it returns email in "email:<email>" format.
// 3. If the subject is available, it returns subject in "subject:<subject>" format.
// 4. Otherwise, it returns EventActorUnknown.
func FormatEventUserActor(u user.Info) string {
	var email, subject, username string
	if u.Username != "" && u.Username != "email" {
		if usernameClaim, ok := u.Claims[u.Username]; ok {
			if usernameStr, ok := usernameClaim.(string); ok {
				username = usernameStr
			}
		}
	}
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
		return kargoapi.EventActorAdmin
	case username != "":
		return kargoapi.EventActorOidcUsernamePrefix + username
	case email != "":
		return kargoapi.EventActorEmailPrefix + email
	case subject != "":
		return kargoapi.EventActorSubjectPrefix + subject
	default:
		return kargoapi.EventActorUnknown
	}
}

func FormatEventKubernetesUserActor(u authnv1.UserInfo) string {
	return kargoapi.EventActorKubernetesUserPrefix + u.Username
}

func NewFreightApprovedEventAnnotations(actor string, f *kargoapi.Freight, stageName string) map[string]string {
	annotations := map[string]string{
		kargoapi.AnnotationKeyEventProject:           f.Namespace,
		kargoapi.AnnotationKeyEventFreightCreateTime: f.CreationTimestamp.Format(time.RFC3339),
		kargoapi.AnnotationKeyEventFreightAlias:      f.Alias,
		kargoapi.AnnotationKeyEventFreightName:       f.Name,
		kargoapi.AnnotationKeyEventStageName:         stageName,
	}
	if actor != "" {
		annotations[kargoapi.AnnotationKeyEventActor] = actor
	}
	return annotations
}
