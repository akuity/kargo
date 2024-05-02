package v1alpha1

const (
	// AnnotationKeyManaged is an annotation key that can be set on a
	// ServiceAccount, Role, or RoleBinding to indicate that it is managed by
	// Kargo.
	AnnotationKeyManaged = "rbac.kargo.akuity.io/managed"

	// AnnotationKeyOIDCEmails is an annotation key that can be set on a
	// ServiceAccount to associate it with a list of email addresses. The
	// annotation is used to grant permissions to the ServiceAccount based on
	// the email claim from a user who authenticates using OIDC.
	// The value of the annotation should be a comma-separated list.
	AnnotationKeyOIDCEmails = "rbac.kargo.akuity.io/email"

	// AnnotationKeyOIDCGroups is an annotation key that can be set on a
	// ServiceAccount to associate it with a list of groups from an OIDC
	// provider. The annotation is used to grant permissions to the
	// ServiceAccount based on the groups claim from a user who authenticates
	// using OIDC.
	// The value of the annotation should be a comma-separated list.
	AnnotationKeyOIDCGroups = "rbac.kargo.akuity.io/groups"

	// AnnotationKeyOIDCSubjects is an annotation key that can be set on a
	// ServiceAccount to associate it with a list of subjects from an OIDC
	// provider. The annotation is used to grant permissions to the
	// ServiceAccount based on the subject claim from a user who authenticates
	// using OIDC.
	// The value of the annotation should be a comma-separated list.
	AnnotationKeyOIDCSubjects = "rbac.kargo.akuity.io/sub"

	AnnotationValueTrue = "true"
)
