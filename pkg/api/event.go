package api

import (
	"fmt"

	authnv1 "k8s.io/api/authentication/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/user"
)

func FormatEventControllerActor(name string) string {
	return kargoapi.EventActorControllerPrefix + name
}

// FormatEventUserActor returns a string representation of the user acting in an event
// that can be used as a value of AnnotationKeyEventActor.
//
//  1. If the user is admin, it returns EventActorAdmin since other information is not available.
//  2. If the user was authenticated by Kubernetes directly (e.g. a Kargo API token),
//     it returns that identity in "kubernetes:<username>" format (see
//     FormatEventKubernetesUserActor).
//  3. If the username is known, it returns username in "<claim>:<username>" format
//     (see formatOIDCUsername).
//  4. If the email is available, it returns email in "email:<email>" format.
//  5. If the subject is available, it returns subject in "subject:<subject>" format.
//  6. Otherwise, it returns EventActorUnknown.
func FormatEventUserActor(u user.Info) string {
	if u.IsAdmin {
		return kargoapi.EventActorAdmin
	}
	if u.KubernetesUserInfo != nil {
		return FormatEventKubernetesUserActor(*u.KubernetesUserInfo)
	}
	if u.Username != "" {
		return formatOIDCUsername(u)
	}
	// Reaching here means the verified token didn't include the claim Kargo is
	// configured to treat as "username." Possible, but unlikely.
	if emailClaim, ok := u.Claims["email"]; ok {
		if email, ok := emailClaim.(string); ok {
			return kargoapi.EventActorEmailPrefix + email
		}
	}
	// Reaching here means the verified token also lacked an "email" claim,
	// even though Kargo always requests the email scope. IdPs aren't
	// guaranteed to send every claim tied to a requested scope. "sub" is
	// always present, so it's the final backstop.
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

func formatOIDCUsername(u user.Info) string {
	return fmt.Sprintf("%s:%s", u.UsernameClaim, u.Username)
}
