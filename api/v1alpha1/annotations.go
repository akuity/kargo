package v1alpha1

const (
	AnnotationKeyDescription = "kargo.akuity.io/description"

	// AnnotationKeyRefresh is an annotation key that can be set on a resource
	// to trigger a refresh of the resource by the controller. The value of the
	// annotation is interpreted as a token, and any change in value should
	// trigger a reconciliation of the resource.
	AnnotationKeyRefresh = "kargo.akuity.io/refresh"

	AnnotationKeyReverify      = "kargo.akuity.io/reverify"
	AnnotationKeyReverifyActor = "kargo.akuity.io/reverify-actor"

	// AnnotationKeyAbort is an annotation key that can be set on a Stage
	// resource to abort the verification of its Freight. The value of the
	// annotation must be set to the identifier of the verification to be
	// aborted.
	AnnotationKeyAbort = "kargo.akuity.io/abort"

	AnnotationKeyOIDCEmails   = "rbac.kargo.akuity.io/email"
	AnnotationKeyOIDCGroups   = "rbac.kargo.akuity.io/groups"
	AnnotationKeyOIDCSubjects = "rbac.kargo.akuity.io/sub"

	AnnotationKeyEventActor                  = "event.kargo.akuity.io/actor"
	AnnotationKeyEventProject                = "event.kargo.akuity.io/project"
	AnnotationKeyEventPromotionName          = "event.kargo.akuity.io/promotion-name"
	AnnotationKeyEventPromotionCreateTime    = "event.kargo.akuity.io/promotion-create-time"
	AnnotationKeyEventFreightAlias           = "event.kargo.akuity.io/freight-alias"
	AnnotationKeyEventFreightName            = "event.kargo.akuity.io/freight-name"
	AnnotationKeyEventFreightCreateTime      = "event.kargo.akuity.io/freight-create-time"
	AnnotationKeyEventFreightCommits         = "event.kargo.akuity.io/freight-commits"
	AnnotationKeyEventFreightImages          = "event.kargo.akuity.io/freight-images"
	AnnotationKeyEventFreightCharts          = "event.kargo.akuity.io/freight-charts"
	AnnotationKeyEventStageName              = "event.kargo.akuity.io/stage-name"
	AnnotationKeyEventAnalysisRunName        = "event.kargo.akuity.io/analysis-run-name"
	AnnotationKeyEventVerificationPending    = "event.kargo.akuity.io/verification-pending"
	AnnotationKeyEventVerificationStartTime  = "event.kargo.akuity.io/verification-start-time"
	AnnotationKeyEventVerificationFinishTime = "event.kargo.akuity.io/verification-finish-time"
)

// RefreshAnnotationValue returns the value of the AnnotationKeyRefresh
// annotation which can be used to detect changes, and a boolean indicating
// whether the annotation was present.
func RefreshAnnotationValue(annotations map[string]string) (string, bool) {
	requested, ok := annotations[AnnotationKeyRefresh]
	return requested, ok
}

// AbortAnnotationValue returns the value of the AnnotationKeyAbort annotation
// which can be used to abort the verification of a Freight, and a boolean
// indicating whether the annotation was present.
func AbortAnnotationValue(annotations map[string]string) (string, bool) {
	requested, ok := annotations[AnnotationKeyAbort]
	return requested, ok
}
