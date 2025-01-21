package v1alpha1

const (
	AliasLabelKey = "kargo.akuity.io/alias"

	// Credentials
	CredentialTypeLabelKey        = "kargo.akuity.io/cred-type" // nolint: gosec
	CredentialTypeLabelValueGit   = "git"
	CredentialTypeLabelValueHelm  = "helm"
	CredentialTypeLabelValueImage = "image"
	CredentialTypeLabelGeneric    = "generic"

	// Project Secrets
	// Deprecated: Use CredentialTypeLabelGeneric instead. This label should not
	// be used and won't be documented, but for now we plan to support it
	// indefinitely for backwards compatibility.
	ProjectSecretLabelKey = "kargo.akuity.io/project-secret" // nolint: gosec

	// Kargo core API
	FreightCollectionLabelKey = "kargo.akuity.io/freight-collection"
	ProjectLabelKey           = "kargo.akuity.io/project"
	PromotionLabelKey         = "kargo.akuity.io/promotion"
	ShardLabelKey             = "kargo.akuity.io/shard"
	StageLabelKey             = "kargo.akuity.io/stage"

	// AnalysisRunTemplate labels
	AnalysisRunTemplateLabelKey         = "kargo.akuity.io/analysis-run-template"
	AnalysisRunTemplateLabelValueConfig = "config"

	LabelTrueValue = "true"

	FinalizerName = "kargo.akuity.io/finalizer"
)
