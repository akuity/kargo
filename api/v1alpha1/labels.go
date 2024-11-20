package v1alpha1

const (
	AliasLabelKey = "kargo.akuity.io/alias"

	// Credentials
	CredentialTypeLabelKey        = "kargo.akuity.io/cred-type" // nolint: gosec
	CredentialTypeLabelValueGit   = "git"
	CredentialTypeLabelValueHelm  = "helm"
	CredentialTypeLabelValueImage = "image"
	// TODO: Should we explicitly keep this label or agree that absence of this label should implicitly say that credential is generic?
	CredentialTypeLabelValueGeneric = "generic"

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
