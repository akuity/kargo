package v1alpha1

const (
	// AliasLabelKey is used to identify the alias of a resource.
	// For example, Freight may have an alias that is easier to remember than
	// the computed name of the resource. For details about the behavior of
	// setting this label, refer to Freight.Alias.
	AliasLabelKey = "kargo.akuity.io/alias"

	// CredentialTypeLabelKey is used to identify the type of credential
	// a Secret contains. For example, a Git credential may be used to access a
	// private repository, while a Helm credential may be used to access a
	// private Helm chart repository.
	CredentialTypeLabelKey = "kargo.akuity.io/cred-type" // nolint: gosec
	// CredentialTypeLabelValueGit is the value for Git credentials.
	// A Secret with this label value is expected to contain credentials for a
	// Git repository.
	CredentialTypeLabelValueGit = "git"
	// CredentialTypeLabelValueHelm is the value for Helm credentials.
	// A Secret with this label value is expected to contain credentials for a
	// Helm repository.
	CredentialTypeLabelValueHelm = "helm"
	// CredentialTypeLabelValueImage is the value for container image registry
	// credentials. A Secret with this label value is expected to contain
	// credentials for a container image registry.
	CredentialTypeLabelValueImage = "image"
	// CredentialTypeLabelValueGeneric is the value for generic credentials.
	// A Secret with this label can contain any type of credential, and is
	// allowed to be managed through the Kargo API.
	CredentialTypeLabelValueGeneric = "generic"

	// StageLabelKey is used to identify the Stage that a resource is associated
	// with. For example, an AnalysisRun created for a specific Stage has this
	// label set to the name of the Stage.
	StageLabelKey = "kargo.akuity.io/stage"
	// FreightCollectionLabelKey is used to identify the FreightCollection
	// that a resource is associated with. For example, an AnalysisRun created
	// for a specific collection of Freight has this label set to the ID of the
	// collection.
	FreightCollectionLabelKey = "kargo.akuity.io/freight-collection"
	// ProjectLabelKey can be used to mark a namespace as a Project namespace
	// by setting the value to "true". This allows Kargo to adopt a namespace
	// that was created before the creation of the Project.
	ProjectLabelKey = "kargo.akuity.io/project"
	// ShardLabelKey is used to identify the shard of a resource.
	ShardLabelKey = "kargo.akuity.io/shard"

	// LabelTrueValue is used to identify a label that has a value of "true".
	LabelTrueValue = "true"

	// FinalizerName is the name of the finalizer used by Kargo.
	FinalizerName = "kargo.akuity.io/finalizer"
)
