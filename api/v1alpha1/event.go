package v1alpha1

const (
	AnnotationKeyEventPrefix                 = "event.kargo.akuity.io/"
	AnnotationKeyEventActor                  = AnnotationKeyEventPrefix + "actor"
	AnnotationKeyEventProject                = AnnotationKeyEventPrefix + "project"
	AnnotationKeyEventPromotionName          = AnnotationKeyEventPrefix + "promotion-name"
	AnnotationKeyEventPromotionCreateTime    = AnnotationKeyEventPrefix + "promotion-create-time"
	AnnotationKeyEventFreightAlias           = AnnotationKeyEventPrefix + "freight-alias"
	AnnotationKeyEventFreightName            = AnnotationKeyEventPrefix + "freight-name"
	AnnotationKeyEventFreightCreateTime      = AnnotationKeyEventPrefix + "freight-create-time"
	AnnotationKeyEventFreightCommits         = AnnotationKeyEventPrefix + "freight-commits"
	AnnotationKeyEventFreightImages          = AnnotationKeyEventPrefix + "freight-images"
	AnnotationKeyEventFreightCharts          = AnnotationKeyEventPrefix + "freight-charts"
	AnnotationKeyEventFreightArtifacts       = AnnotationKeyEventPrefix + "freight-artifacts"
	AnnotationKeyEventStageName              = AnnotationKeyEventPrefix + "stage-name"
	AnnotationKeyEventAnalysisRunName        = AnnotationKeyEventPrefix + "analysis-run-name"
	AnnotationKeyEventVerificationPending    = AnnotationKeyEventPrefix + "verification-pending"
	AnnotationKeyEventVerificationStartTime  = AnnotationKeyEventPrefix + "verification-start-time"
	AnnotationKeyEventVerificationFinishTime = AnnotationKeyEventPrefix + "verification-finish-time"
	AnnotationKeyEventApplications           = AnnotationKeyEventPrefix + "applications"
)

const (
	EventTypePromotionCreated                EventType = "PromotionCreated"
	EventTypePromotionSucceeded              EventType = "PromotionSucceeded"
	EventTypePromotionFailed                 EventType = "PromotionFailed"
	EventTypePromotionErrored                EventType = "PromotionErrored"
	EventTypePromotionAborted                EventType = "PromotionAborted"
	EventTypeFreightApproved                 EventType = "FreightApproved"
	EventTypeFreightVerificationSucceeded    EventType = "FreightVerificationSucceeded"
	EventTypeFreightVerificationFailed       EventType = "FreightVerificationFailed"
	EventTypeFreightVerificationErrored      EventType = "FreightVerificationErrored"
	EventTypeFreightVerificationAborted      EventType = "FreightVerificationAborted"
	EventTypeFreightVerificationInconclusive EventType = "FreightVerificationInconclusive"
	EventTypeFreightVerificationUnknown      EventType = "FreightVerificationUnknown"
)

const (
	EventActorAdmin                = "admin"
	EventActorControllerPrefix     = "controller:"
	EventActorEmailPrefix          = "email:"
	EventActorSubjectPrefix        = "subject:"
	EventActorKubernetesUserPrefix = "kubernetes:"
	EventActorUnknown              = "unknown actor"
)

type EventType string
