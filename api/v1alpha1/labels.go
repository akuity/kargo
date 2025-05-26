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
	// migrated, indicating that no further actions are required. The label is
	// set to one or more values to indicate the type of migration that has
	// been performed.
	MigratedLabelKey = "kargo.akuity.io/migrated"
	// MigratedLabelValueProjectSpec is the value of the MigratedLabelKey that
	// indicates that the ProjectSpec has been migrated to a ProjectConfig
	// resource.
	MigratedLabelValueProjectSpec = "project-spec"

	LabelTrueValue = "true"

	FinalizerName = "kargo.akuity.io/finalizer"
)
