package v1alpha1

const (
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
	AnnotationKeyEventApplications           = "event.kargo.akuity.io/applications"
)

const (
	EventReasonPromotionCreated                = "PromotionCreated"
	EventReasonPromotionSucceeded              = "PromotionSucceeded"
	EventReasonPromotionFailed                 = "PromotionFailed"
	EventReasonPromotionErrored                = "PromotionErrored"
	EventReasonPromotionAborted                = "PromotionAborted"
	EventReasonFreightApproved                 = "FreightApproved"
	EventReasonFreightVerificationSucceeded    = "FreightVerificationSucceeded"
	EventReasonFreightVerificationFailed       = "FreightVerificationFailed"
	EventReasonFreightVerificationErrored      = "FreightVerificationErrored"
	EventReasonFreightVerificationAborted      = "FreightVerificationAborted"
	EventReasonFreightVerificationInconclusive = "FreightVerificationInconclusive"
	EventReasonFreightVerificationUnknown      = "FreightVerificationUnknown"
)

const (
	EventActorAdmin                = "admin"
	EventActorControllerPrefix     = "controller:"
	EventActorEmailPrefix          = "email:"
	EventActorSubjectPrefix        = "subject:"
	EventActorKubernetesUserPrefix = "kubernetes:"
	EventActorUnknown              = "unknown actor"
)

var EventActorOIDCClaimPrefix = os.GetEnv("OIDC_USERNAME_CLAIM", "email") + ":"
