package v1alpha1

const (
	AnnotationKeyDescription = "kargo.akuity.io/description"

	AnnotationKeyRefresh = "kargo.akuity.io/refresh"

	AnnotationKeyReverify      = "kargo.akuity.io/reverify"
	AnnotationKeyReverifyActor = "kargo.akuity.io/reverify-actor"
	AnnotationKeyAbort         = "kargo.akuity.io/abort"

	AnnotationKeyOIDCEmails   = "rbac.kargo.akuity.io/email"
	AnnotationKeyOIDCGroups   = "rbac.kargo.akuity.io/groups"
	AnnotationKeyOIDCSubjects = "rbac.kargo.akuity.io/sub"

	AnnotationKeyEventActor               = "event.kargo.akuity.io/actor"
	AnnotationKeyEventProject             = "event.kargo.akuity.io/project"
	AnnotationKeyEventPromotionName       = "event.kargo.akuity.io/promotion-name"
	AnnotationKeyEventPromotionCreateTime = "event.kargo.akuity.io/promotion-create-time"
	AnnotationKeyEventFreightAlias        = "event.kargo.akuity.io/freight-alias"
	AnnotationKeyEventFreightName         = "event.kargo.akuity.io/freight-name"
	AnnotationKeyEventStageName           = "event.kargo.akuity.io/stage-name"
	AnnotationKeyEventAnalysisRunName     = "event.kargo.akuity.io/analysis-run-name"
)
