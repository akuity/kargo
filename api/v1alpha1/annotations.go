package v1alpha1

const (
	// AnnotationKeyCreateActor is an annotation key that can be injected to a
	// resource by the Kargo control plane to indicate the actor that created
	// the resource.
	AnnotationKeyCreateActor = "kargo.akuity.io/create-actor"

	// AnnotationKeyRefresh is an annotation key that can be set on a resource
	// to trigger a refresh of the resource by the controller. The value of the
	// annotation is interpreted as a token, and any change to the value of the
	// annotation should trigger a reconciliation of the resource.
	AnnotationKeyRefresh = "kargo.akuity.io/refresh"

	// AnnotationKeyReverify is an annotation key that can be set on a Stage
	// resource to trigger the re-verification of its Freight. The value of the
	// annotation should either be the ID of the verification to be reverified,
	// or a JSON object with the structure of the VerificationRequest.
	AnnotationKeyReverify = "kargo.akuity.io/reverify"

	// AnnotationKeyAbort is an annotation key that can be set on a Stage
	// resource to abort the verification of its Freight. The value of the
	// annotation must be set to the identifier of the verification to be
	// aborted.
	AnnotationKeyAbort = "kargo.akuity.io/abort"

	// AnnotationKeyDescription is an annotation key that can be set on a
	// resource to provide a description of it. The value of the annotation may
	// be used by the Kargo UI to display additional information about the
	// resource.
	AnnotationKeyDescription = "kargo.akuity.io/description"

	// AnnotationKeyAuthorizedStage is an annotation key that can be set on a
	// resource to indicate that a Stage is authorized to manage it. The value
	// of the annotation should be in the format of "<project>:<stage>".
	AnnotationKeyAuthorizedStage = "kargo.akuity.io/authorized-stage"

	// AnnotationKeyStage is an annotation key that can be set on a resource to
	// indicate that it is associated with a specific Stage. It compliments
	// LabelKeyStage, which may contain the same value but in a (hash-)shortened
	// form to fit within the Kubernetes label value length. The value of this
	// annotation is expected to be the full name of the Stage.
	AnnotationKeyStage = "kargo.akuity.io/stage"

	// AnnotationKeyPromotion is an annotation key that can be set on a
	// resource to indicate that it is related to a specific promotion.
	AnnotationKeyPromotion = "kargo.akuity.io/promotion"

	// AnnotationKeyArgoCDContext is an annotation key that is set on a Stage
	// to reference the last ArgoCD Applications that were part of a Promotion.
	AnnotationKeyArgoCDContext = "kargo.akuity.io/argocd-context"

	// AnnotationKeyMigrated is an annotation set on a resource that has
	// successfully undergone a migration to a new resource type or other
	// configuration change. This annotation is used to indicate that the
	// resource has been successfully migrated and that the controller should
	// not attempt to perform the migration again.
	//
	// The value of the annotation is a JSON object that maps migration types to
	// booleans indicating whether the migration has been performed.
	AnnotationKeyMigrated = "kargo.akuity.io/migrated"

	// AnnotationKeyKeepNamespace is an annotation key that can be set on Project
	// or Namespace to disable the automatic deletion of the namespace when
	// the Project is deleted. This is useful for cases where the namespace
	// contains resources that should not be deleted.
	AnnotationKeyKeepNamespace = "kargo.akuity.io/keep-namespace"

	// AnnotationValueTrue is the value used to indicate that an annotation
	// is set to true.
	AnnotationValueTrue = "true"

	// AnnotationKeyGitHubTokenScope is the key for an annotation that can
	// optionally be added to any Secret resources that represents a GitHub App
	// installation in order to limit the scope of the installation access tokens
	// that are issued as-needed to specific Kargo Projects.
	//
	// If present, the annotation's value must be a string representation of a
	// JSON object mapping Project names to lists of allowed scopes (repository
	// names).
	//
	// For example to limit tokens issued to Project kargo-demo-1 to scopes repo-a
	// and repo-b only and limit tokens issued to kargo-demo-2 to scope repo-c
	// only:
	//
	//   `{"kargo-demo-1": ["repo-a", "repo-b"], "kargo-demo-2": ["repo-c"]}`
	//
	// ALL OTHER PROJECTS WOULD EFFECTIVELY BE UNABLE TO OBTAIN AN INSTALLATION
	// TOKEN AT ALL.
	//
	// This annotation has an effect only when present. i.e. If not present, the
	// scopes available to every Project are unconstrained. If the annotation is
	// present, with an invalid value (not well-formed JSON), NO Project will be
	// able to obtain an installation token.
	//
	// #nosec G101 -- This is not a credential, just an annotation key name.
	AnnotationKeyGitHubTokenScope = "kargo.akuity.io/github-token-scopes"
)
