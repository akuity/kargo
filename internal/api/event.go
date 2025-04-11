package api

import (
	"fmt"
	"os"
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
	if u.IsAdmin {
		return kargoapi.EventActorAdmin
	}
	if u.Username != "" {
		return formatOidcUsername(u.Username)
	}
	if emailClaim, ok := u.Claims["email"]; ok {
		if email, ok := emailClaim.(string); ok {
			return kargoapi.EventActorEmailPrefix + email
		}
	}
	if subClaim, ok := u.Claims["sub"]; ok {
		if sub, ok := subClaim.(string); ok {
			return kargoapi.EventActorSubjectPrefix + sub
		}
	}
	return kargoapi.EventActorUnknown
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

func formatOidcUsername(oidcUsername string) string {
	return fmt.Sprintf("%s:%s", os.Getenv("OIDC_USERNAME_CLAIM"), oidcUsername)
}
