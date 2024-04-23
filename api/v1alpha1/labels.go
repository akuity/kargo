package v1alpha1

const (
	AliasLabelKey = "kargo.akuity.io/alias"

	// Credentials
	CredentialTypeLabelKey        = "kargo.akuity.io/cred-type" // nolint: gosec
	CredentialTypeLabelValueGit   = "git"
	CredentialTypeLabelValueHelm  = "helm"
	CredentialTypeLabelValueImage = "image"

	// Kargo core API
	FreightLabelKey   = "kargo.akuity.io/freight"
	ProjectLabelKey   = "kargo.akuity.io/project"
	PromotionLabelKey = "kargo.akuity.io/promotion"
	ShardLabelKey     = "kargo.akuity.io/shard"
	StageLabelKey     = "kargo.akuity.io/stage"

	LabelTrueValue = "true"

	FinalizerName = "kargo.akuity.io/finalizer"
)
