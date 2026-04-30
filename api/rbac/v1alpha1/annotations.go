package v1alpha1

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
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
	// A string representation of a JSON object in the AnnotationKeyOIDCClaims
	// is the preferred method.
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

// stringOrArray is a helper type that can unmarshal from either a scalar string
// or an array of strings.
type stringOrArray []string

func (s *stringOrArray) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = arr
		return nil
	}
	// Fall back to scalar string
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = []string{str}
	return nil
}

// OIDCClaimsFromAnnotationValue parses the values of the newer, preferred
// rbac.kargo.akuity.io/claims annotation as well as the values of older
// annotations with keys of the form rbac.kargo.akuity.io/claim.<name> and
// consolidates them into a single map of claim names to (deduped) claim values
// describing the set of users mapped to some ServiceAccount.
func OIDCClaimsFromAnnotationValues(annotations map[string]string) (map[string][]string, error) {
	claims := make(map[string][]string)
	// hydrate with new style claims
	if _, ok := annotations[AnnotationKeyOIDCClaims]; ok {
		flexClaims := map[string]stringOrArray{}
		if err := json.Unmarshal([]byte(annotations[AnnotationKeyOIDCClaims]), &flexClaims); err != nil {
			return nil, fmt.Errorf("unmarshaling OIDC claims from annotation value: %w", err)
		}
		for key, val := range flexClaims {
			claims[key] = []string(val)
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

// SetOIDCClaimsAnnotation marshals the provided map of claims to claim values
// and writes its string representation as the value of the ServiceAccount's
// rbac.kargo.akuity.io/claims annotation, which is the newer, preferred method
// of mapping users to ServiceAccounts. This function also removes
// any annotations with keys of the form rbac.kargo.akuity.io/claim.<name> (the
// older method) from the ServiceAccount.
func SetOIDCClaimsAnnotation(sa *corev1.ServiceAccount, claims map[string][]string) error {
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return fmt.Errorf("marshaling OIDC claims to annotation value: %w", err)
	}
	if sa.Annotations == nil {
		sa.Annotations = map[string]string{}
	}
	sa.Annotations[AnnotationKeyOIDCClaims] = string(claimsJSON)
	for k := range sa.Annotations {
		if strings.HasPrefix(k, AnnotationKeyOIDCClaimNamePrefix) {
			delete(sa.Annotations, k)
		}
	}
	return nil
}
