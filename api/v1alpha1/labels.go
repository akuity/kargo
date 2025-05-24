package v1alpha1

const (
	// LabelKeyAlias is used to identify the alias of a resource.
	// For example, Freight may have an alias that is easier to remember than
	// the computed name of the resource. For details about the behavior of
	// setting this label, refer to Freight.Alias.
	LabelKeyAlias = "kargo.akuity.io/alias"

	// LabelKeyCredentialType is used to identify the type of credential
	// a Secret contains. For example, a Git credential may be used to access a
	// private repository, while a Helm credential may be used to access a
	// private Helm chart repository.
	LabelKeyCredentialType = "kargo.akuity.io/cred-type" // nolint: gosec
	// LabelValueCredentialTypeGit is the value for Git credentials.
	// A Secret with this label value is expected to contain credentials for a
	// Git repository.
	LabelValueCredentialTypeGit = "git"
	// LabelValueCredentialTypeHelm is the value for Helm credentials.
	// A Secret with this label value is expected to contain credentials for a
	// Helm repository.
	LabelValueCredentialTypeHelm = "helm"
	// LabelValueCredentialTypeImage is the value for container image registry
	// credentials. A Secret with this label value is expected to contain
	// credentials for a container image registry.
	LabelValueCredentialTypeImage = "image"
	// LabelValueCredentialTypeGeneric is the value for generic credentials.
	// A Secret with this label can contain any type of credential, and is
	// allowed to be managed through the Kargo API.
	LabelValueCredentialTypeGeneric = "generic"

	// LabelKeyStage is used to identify the Stage that a resource is associated
	// with. For example, an AnalysisRun created for a specific Stage has this
	// label set to the name of the Stage.
	LabelKeyStage = "kargo.akuity.io/stage"
	// LabelKeyFreightCollection is used to identify the FreightCollection
	// that a resource is associated with. For example, an AnalysisRun created
	// for a specific collection of Freight has this label set to the ID of the
	// collection.
	LabelKeyFreightCollection = "kargo.akuity.io/freight-collection"
	// LabelKeyProject can be used to mark a namespace as a Project namespace
	// by setting the value to "true". This allows Kargo to adopt a namespace
	// that was created before the creation of the Project.
	LabelKeyProject = "kargo.akuity.io/project"
	// LabelKeyShard is used to identify the shard of a resource.
	LabelKeyShard         = "kargo.akuity.io/shard"

	// LabelValueTrue is used to identify a label that has a value of "true".
	LabelValueTrue = "true"

	// FinalizerName is the name of the finalizer used by Kargo.
	FinalizerName = "kargo.akuity.io/finalizer"
)
