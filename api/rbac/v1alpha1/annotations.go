package v1alpha1

import "strings"

const (
	// AnnotationKeyManaged is an annotation key that can be set on a
	// ServiceAccount, Role, or RoleBinding to indicate that it is managed by
	// Kargo.
	AnnotationKeyManaged = "rbac.kargo.akuity.io/managed"

	// AnnotationKeyOIDCClaimNamePrefix is the prefix of an annotation key that
	// can be set on a ServiceAccount to associate it with any user authenticated
	// via OIDC and having the claim indicated by the full annotation key with
	// any of the values indicated by the annotation. The value of the annotation
	// may be either a scalar string value or a comma-separated list.
	AnnotationKeyOIDCClaimNamePrefix = "rbac.kargo.akuity.io/claim."

	// AnnotationKeyOIDCClaims is an annotation key that can be set on a
	// ServiceAccount to associate it with any user authenticated via OIDC and
	// having any of the claims indicated by the value of the annotation. The
	// value is expected to be a map[string]string and is intended for use
	// when AnnotationKeyOIDCClaimNamePrefix is not possible to use due to
	// to the misrepresentation of special characters.
	AnnotationKeyOIDCClaims = "rbac.kargo.akuity.io/claims"

	// AnnotationValueTrue is a value that can be set on an annotation to indicate
	// that it applies.
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

func AnnotationKeyOIDCClaimsValue(keyValuePairs ...string) string {
	return ""
}
