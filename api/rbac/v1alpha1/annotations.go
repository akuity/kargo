package v1alpha1

const (
	// AnnotationKeyManaged is an annotation key that can be set on a
	// ServiceAccount, Role, or RoleBinding to indicate that it is managed by
	// Kargo.
	AnnotationKeyManaged = "rbac.kargo.akuity.io/managed"

	// AnnotationKeyOIDCPrefix is an annotation key prefix that can be set on a
	// ServiceAccount to associate it with a list of claims from an OIDC
	// provider. The annotation is used to grant permissions to the
	// ServiceAccount based on the claim that is both provided in the suffix and from a user who authenticates
	// using OIDC.
	// e.g. rbac.kargo.akuity.io/claim.subs: '123,456'
	// The value of the annotation should be a comma-separated list.
	AnnotationKeyOIDCClaimNamePrefix = "rbac.kargo.akuity.io/claim."

	AnnotationValueTrue = "true"
)
