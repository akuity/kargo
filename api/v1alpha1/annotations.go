package v1alpha1

const (
	AnnotationKeyDescription = "kargo.akuity.io/description"

	AnnotationKeyRefresh = "kargo.akuity.io/refresh"

	AnnotationKeyReverify = "kargo.akuity.io/reverify"
	AnnotationKeyAbort    = "kargo.akuity.io/abort"

	AnnotationKeyOIDCEmails   = "rbac.kargo.akuity.io/email"
	AnnotationKeyOIDCGroups   = "rbac.kargo.akuity.io/groups"
	AnnotationKeyOIDCSubjects = "rbac.kargo.akuity.io/sub"

	AnnotationKeyEventActor                    = "event.kargo.akuity.io/actor"
	AnnotationKeyEventProject                  = "event.kargo.akuity.io/project"
	AnnotationKeyEventPromotionName            = "event.kargo.akuity.io/promotion-name"
	AnnotationKeyEventPromotionCreateTime      = "event.kargo.akuity.io/promotion-create-time"
	AnnotationKeyEventFreightAlias             = "event.kargo.akuity.io/freight-alias"
	AnnotationKeyEventFreightName              = "event.kargo.akuity.io/freight-name"
	AnnotationKeyEventFreightCreateTime        = "event.kargo.akuity.io/freight-create-time"
	AnnotationKeyEventFreightCommits           = "event.kargo.akuity.io/freight-commits"
	AnnotationKeyEventFreightImages            = "event.kargo.akuity.io/freight-images"
	AnnotationKeyEventFreightCharts            = "event.kargo.akuity.io/freight-charts"
	AnnotationKeyEventStageName                = "event.kargo.akuity.io/stage-name"
	AnnotationKeyEventAnalysisRunName          = "event.kargo.akuity.io/analysis-run-name"
	AnnotationKeyEventVerificationPending      = "event.kargo.akuity.io/verification-pending"
	AnnotationKeyEventVerificationStartTime    = "event.kargo.akuity.io/verification-start-time"
	AnnotationKeyEventVerificationCompleteTime = "event.kargo.akuity.io/verification-complete-time"
)
