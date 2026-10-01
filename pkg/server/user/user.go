// Package user describes who a request to the Kargo API server is from.
//
// The authentication middleware resolves a bearer token to an Identity and
// binds it to the request context. Handlers and the authorizing Kubernetes
// client read it back to decide what the request may do and to record who
// did it. Each kind of Identity knows how it is authorized, so nothing else
// in the server has to tell them apart.
package user

import (
	"context"
	"fmt"

	authnv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Identity is who a request is from.
type Identity interface {
	// IsAdmin reports whether this is the Kargo API server's own admin user,
	// who may do everything the server can do and is never reviewed.
	IsAdmin() bool
	// Subjects returns the Kubernetes subjects whose permissions stand in
	// for this identity when performing the described operation. The
	// operation is allowed if any one of them is allowed. An identity with
	// no subjects is allowed nothing.
	Subjects(ra authv1.ResourceAttributes) []Subject
	// Actor names the identity the way events and audit annotations record
	// it.
	Actor() string
	// MemberOf reports whether the identity has a presence in the Project
	// namespace, such that the Project counts as one of theirs. It implies no
	// permission in it. Only an OIDCUser, through a mapped ServiceAccount,
	// ever does; nothing is recorded about which Projects anyone else belongs
	// to.
	MemberOf(namespace string) bool
}

var (
	_ Identity = Admin{}
	_ Identity = OIDCUser{}
	_ Identity = KubernetesUser{}
)

// Subject is a Kubernetes identity whose access can be reviewed.
type Subject struct {
	Username string
	UID      string
	Groups   []string
	Extra    map[string]authv1.ExtraValue
}

// Admin is the Kargo API server's own admin user, authenticated by a token
// the server itself issued.
type Admin struct{}

func (Admin) IsAdmin() bool { return true }

// Subjects returns nothing: an Admin is allowed without review.
func (Admin) Subjects(authv1.ResourceAttributes) []Subject { return nil }

func (Admin) Actor() string { return kargoapi.EventActorAdmin }

// MemberOf returns false: an Admin belongs to every Project, so none is
// singled out as theirs.
func (Admin) MemberOf(string) bool { return false }

// OIDCUser is a user authenticated by Kargo's OpenID Connect identity
// provider. Such a user holds no Kubernetes identity of their own; they act
// through the ServiceAccounts their claims map them to.
type OIDCUser struct {
	// Claims are the verified claims of the user's token.
	Claims map[string]any
	// UsernameClaim names the claim the Username was taken from.
	UsernameClaim string
	// Username is the user's name as the configured claim states it. It is
	// often an email address, but may be anything the identity provider
	// sends.
	Username string
	// ServiceAccountsByNamespace maps namespaces to the ServiceAccounts in
	// them that the user's claims map to.
	ServiceAccountsByNamespace map[string]map[types.NamespacedName]struct{}
	// GlobalServiceAccountNamespaces are the namespaces, besides a Project's
	// own, in which a mapped ServiceAccount may grant access to the Project.
	GlobalServiceAccountNamespaces []string
}

func (OIDCUser) IsAdmin() bool { return false }

// Subjects returns the user's mapped ServiceAccounts in the namespaces that
// may grant the operation: the resource's own namespace first, where a
// ServiceAccount with the required permissions is most likely to be found,
// then the global ServiceAccount namespaces. A cluster-scoped Project is the
// one resource whose own namespace is its name.
func (u OIDCUser) Subjects(ra authv1.ResourceAttributes) []Subject {
	namespaces := make([]string, 0, len(u.GlobalServiceAccountNamespaces)+1)
	switch {
	case ra.Namespace != "":
		namespaces = append(namespaces, ra.Namespace)
	case ra.Group == kargoapi.GroupVersion.Group && ra.Resource == "projects":
		namespaces = append(namespaces, ra.Name)
	}
	namespaces = append(namespaces, u.GlobalServiceAccountNamespaces...)
	var subjects []Subject
	for _, namespace := range namespaces {
		for sa := range u.ServiceAccountsByNamespace[namespace] {
			subjects = append(subjects, ServiceAccountSubject(sa))
		}
	}
	return subjects
}

// MemberOf reports whether the user's claims map them to at least one
// ServiceAccount in the namespace.
func (u OIDCUser) MemberOf(namespace string) bool {
	return len(u.ServiceAccountsByNamespace[namespace]) > 0
}

// Actor names the user by the configured username claim when the token
// carried it, else by email, else by subject. Identity providers are not
// guaranteed to send every claim tied to a requested scope, but "sub" is
// always present, so it is the final backstop.
func (u OIDCUser) Actor() string {
	if u.Username != "" {
		return fmt.Sprintf("%s:%s", u.UsernameClaim, u.Username)
	}
	if email, ok := u.Claims["email"].(string); ok {
		return kargoapi.EventActorEmailPrefix + email
	}
	if sub, ok := u.Claims["sub"].(string); ok {
		return kargoapi.EventActorSubjectPrefix + sub
	}
	return kargoapi.EventActorUnknown
}

// KubernetesUser is the holder of a bearer token that Kubernetes itself
// recognized, through a TokenReview: a ServiceAccount, such as a Kargo API
// token or a controller, or a user of the cluster's own identity provider.
type KubernetesUser struct {
	authnv1.UserInfo
}

func (KubernetesUser) IsAdmin() bool { return false }

// Subjects returns the user as Kubernetes reported them. Groups, UID and
// extras are all carried across, because omitting them would ask a different
// question: whether the user is permitted on the strength of their username
// alone.
func (u KubernetesUser) Subjects(authv1.ResourceAttributes) []Subject {
	subject := Subject{
		Username: u.Username,
		UID:      u.UID,
		Groups:   u.Groups,
	}
	if len(u.Extra) > 0 {
		subject.Extra = make(map[string]authv1.ExtraValue, len(u.Extra))
		for k, v := range u.Extra {
			subject.Extra[k] = authv1.ExtraValue(v)
		}
	}
	return []Subject{subject}
}

func (u KubernetesUser) Actor() string {
	return kargoapi.EventActorKubernetesUserPrefix + u.Username
}

// MemberOf returns false: Kubernetes vouches for who the user is, not for
// which Projects are theirs.
func (KubernetesUser) MemberOf(string) bool { return false }

// ServiceAccountSubject is the subject a ServiceAccount is reviewed as.
func ServiceAccountSubject(name types.NamespacedName) Subject {
	return Subject{
		Username: fmt.Sprintf("system:serviceaccount:%s:%s", name.Namespace, name.Name),
	}
}

type identityKey struct{}

// ContextWithIdentity returns a context carrying the Identity.
func ContextWithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// IdentityFromContext returns the Identity bound to the context, if any.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok && id != nil
}
