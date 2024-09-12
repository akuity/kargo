package v1alpha1

import strings "strings"

const (
	// AnnotationKeyManaged is an annotation key that can be set on a
	// ServiceAccount, Role, or RoleBinding to indicate that it is managed by
	// Kargo.
	AnnotationKeyManaged = "rbac.kargo.akuity.io/managed"

	// AnnotationKeyOIDCPrefix is the prefix of an annotation key that can be set on a
	// ServiceAccount to associate it with any user authenticated via OIDC and having
	// the claim indicated by the full annotation key with any of the values indicated by
	// the annotation. The value of the annotation may be either a scalar string value or a
	// comma-separated list.
	AnnotationKeyOIDCClaimNamePrefix = "rbac.kargo.akuity.io/claim."

	AnnotationValueTrue = "true"
)

func AnnotationKeyOIDCClaim(name string) string {
	return AnnotationKeyOIDCClaimNamePrefix + name
}

func OIDCClaimNameFromAnnotationKey(key string) (string, bool) {
	if !strings.HasPrefix(key, AnnotationKeyOIDCClaimNamePrefix) {
		return "", false
	}
	return strings.TrimPrefix(key, AnnotationKeyOIDCClaimNamePrefix), true
}
