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

	// AnnotationTrueValue is the value used to indicate that an annotation
	// is set to true.
	AnnotationTrueValue = "true"
)
