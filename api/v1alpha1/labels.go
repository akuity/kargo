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

	// MigratedLabelKey is a label set on a resource that has successfully been
	// migrated, indicating that no further actions are required. This label is
	// set to "true" when the migration is complete. An example of this is the
	// migration of the ProjectSpec to a ProjectConfig resource.
	MigratedLabelKey = "kargo.akuity.io/migrated"

	LabelTrueValue = "true"

	FinalizerName = "kargo.akuity.io/finalizer"
)
