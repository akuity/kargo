package v1alpha1

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

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
	// value is expected to be a string representation of a JSON object containing
	// claim names as keys mapped to claim values represented as lists of strings.
	//
	// For example:
	//
	//   `{"email": ["kilgore@kilgore.trout"], "groups": ["devops", "maintainers"]}`
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

// OIDCClaimsFromAnnotationValue parses the value of an
// rbac.kargo.akuity.io/claims annotation and returns the corresponding list of
// formatted and sorted claims.
func OIDCClaimsFromAnnotationValue(value string) ([]string, error) {
	claims := make(map[string][]string)
	if err := json.Unmarshal([]byte(value), &claims); err != nil {
		return nil, fmt.Errorf("unmarshaling OIDC claims from annotation value: %w", err)
	}
	var result []string
	for name, values := range claims {
		for _, v := range values {
			result = append(result,
				FormatClaim(name, strings.TrimSpace(v)),
			)
		}
	}
	return result, nil
}

// OIDCClaimsFromAnnotationValue parses the values of both the
// rbac.kargo.akuity.io/claims and rbac.kargo.akuity.io/claim.<name> annotations
// and consolidates them into a single map where the value of each key is sorted and deduped.
func OIDCClaimsFromAnnotationValues(annotations map[string]string) (map[string][]string, error) {
	claims := make(map[string][]string)
	// hydrate with new style claims
	if _, ok := annotations[AnnotationKeyOIDCClaims]; ok {
		if err := json.Unmarshal([]byte(annotations[AnnotationKeyOIDCClaims]), &claims); err != nil {
			return nil, fmt.Errorf("unmarshaling OIDC claims from annotation value: %w", err)
		}
	}
	// hydrate with old style claims
	for name, values := range annotations {
		if key, ok := OIDCClaimNameFromAnnotationKey(name); ok {
			for v := range strings.SplitSeq(values, ",") {
				claims[key] = append(claims[key], strings.TrimSpace(v))
			}
		}
	}
	for k, v := range claims {
		slices.Sort(v)
		claims[k] = slices.Compact(v)
	}
	return claims, nil
}

// FormatClaim formats a claims name and values to be used by the
// IndexServiceAccountsByOIDCClaims index.
func FormatClaim(claimName, claimValue string) string {
	return claimName + "/" + claimValue
}
