package v1alpha1

const (
	AliasLabelKey = "kargo.akuity.io/alias"

	// Credentials
	CredentialTypeLabelKey        = "kargo.akuity.io/cred-type" // nolint: gosec
	CredentialTypeLabelValueGit   = "git"
	CredentialTypeLabelValueHelm  = "helm"
	CredentialTypeLabelValueImage = "image"
	CredentialTypeLabelGeneric    = "generic"

	// Kargo core API
	FreightCollectionLabelKey = "kargo.akuity.io/freight-collection"
	ProjectLabelKey           = "kargo.akuity.io/project"
	ShardLabelKey             = "kargo.akuity.io/shard"
	StageLabelKey             = "kargo.akuity.io/stage"

	LabelTrueValue = "true"

	FinalizerName = "kargo.akuity.io/finalizer"
)
