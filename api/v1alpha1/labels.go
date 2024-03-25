package v1alpha1

const (
	AliasLabelKey = "kargo.akuity.io/alias"

	CredentialTypeLabelKey        = "kargo.akuity.io/cred-type" // nolint: gosec
	CredentialTypeLabelValueGit   = "git"
	CredentialTypeLabelValueHelm  = "helm"
	CredentialTypeLabelValueImage = "image"

	FreightLabelKey = "kargo.akuity.io/freight"
	ProjectLabelKey = "kargo.akuity.io/project"
	ShardLabelKey   = "kargo.akuity.io/shard"
	StageLabelKey   = "kargo.akuity.io/stage"

	LabelTrueValue = "true"

	FinalizerName = "kargo.akuity.io/finalizer"

	AllowSharedOwnershipLabelKey = "kargo.akuity.io/shared-owner"
)
