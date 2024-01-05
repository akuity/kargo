package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:validation:Enum={Digest,Lexical,NewestBuild,SemVer}
type ImageTagSelectionStrategy string

const (
	ImageTagSelectionStrategyDigest      ImageTagSelectionStrategy = "Digest"
	ImageTagSelectionStrategyLexical     ImageTagSelectionStrategy = "Lexical"
	ImageTagSelectionStrategyNewestBuild ImageTagSelectionStrategy = "NewestBuild"
	ImageTagSelectionStrategySemVer      ImageTagSelectionStrategy = "SemVer"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Warehouse is a source of Freight.
type Warehouse struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes sources of artifacts.
	//
	//+kubebuilder:validation:Required
	Spec *WarehouseSpec `json:"spec"`
	// Status describes the Warehouse's most recently observed state.
	Status WarehouseStatus `json:"status,omitempty"`
}

func (w *Warehouse) GetStatus() *WarehouseStatus {
	return &w.Status
}

// WarehouseSpec describes sources of versioned artifacts to be included in
// Freight produced by this Warehouse.
type WarehouseSpec struct {
	// Subscriptions describes sources of artifacts to be included in Freight
	// produced by this Warehouse.
	//
	//+kubebuilder:validation:MinItems=1
	Subscriptions []RepoSubscription `json:"subscriptions"`
}

// RepoSubscription describes a subscription to ONE OF a Git repository, a
// container image repository, or a Helm chart repository.
type RepoSubscription struct {
	// Git describes a subscriptions to a Git repository.
	Git *GitSubscription `json:"git,omitempty"`
	// Image describes a subscription to container image repository.
	Image *ImageSubscription `json:"image,omitempty"`
	// Chart describes a subscription to a Helm chart repository.
	Chart *ChartSubscription `json:"chart,omitempty"`
}

// GitSubscription defines a subscription to a Git repository.
type GitSubscription struct {
	// URL is the repository's URL. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^https://(\w+([\.-]\w+)*@)?\w+([\.-]\w+)*(:[\d]+)?(/.*)?$`
	RepoURL string `json:"repoURL"`
	// Branch references a particular branch of the repository. This field is
	// optional. When not specified, the subscription is implicitly to the
	// repository's default branch.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^\w+([-/]\w+)*$`
	Branch string `json:"branch,omitempty"`
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when connecting to the repository. This should be enabled
	// only with great caution.
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

// ImageSubscription defines a subscription to an image repository.
type ImageSubscription struct {
	// RepoURL specifies the URL of the image repository to subscribe to. The
	// value in this field MUST NOT include an image tag. This field is required.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(\w+([\.-]\w+)*(:[\d]+)?/)?(\w+([\.-]\w+)*)(/\w+([\.-]\w+)*)*$`
	RepoURL string `json:"repoURL"`
	// GitRepoURL optionally specifies the URL of a Git repository that contains
	// the source code for the image repository referenced by the RepoURL field.
	// When this is specified, Kargo MAY be able to infer and link to the exact
	// revision of that source code that was used to build the image.
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=`^https://(\w+([\.-]\w+)*@)?\w+([\.-]\w+)*(:[\d]+)?(/.*)?$`
	GitRepoURL string `json:"gitRepoURL,omitempty"`
	// TagSelectionStrategy specifies the rules for how to identify the newest version
	// of the image specified by the RepoURL field. This field is optional. When
	// left unspecified, the field is implicitly treated as if its value were
	// "SemVer".
	//
	// +kubebuilder:default=SemVer
	TagSelectionStrategy ImageTagSelectionStrategy `json:"tagSelectionStrategy,omitempty"`
	// SemverConstraint specifies constraints on what new image versions are
	// permissible. The value in this field only has any effect when the
	// TagSelectionStrategy is SemVer or left unspecified (which is implicitly the
	// same as SemVer). This field is also optional. When left unspecified, (and
	// the TagSelectionStrategy is SemVer or unspecified), there will be no
	// constraints, which means the latest semantically tagged version of an image
	// will always be used. Care should be taken with leaving this field
	// unspecified, as it can lead to the unanticipated rollout of breaking
	// changes. Refer to Image Updater documentation for more details.
	//
	//+kubebuilder:validation:Optional
	SemverConstraint string `json:"semverConstraint,omitempty"`
	// AllowTags is a regular expression that can optionally be used to limit the
	// image tags that are considered in determining the newest version of an
	// image. This field is optional.
	//
	//+kubebuilder:validation:Optional
	AllowTags string `json:"allowTags,omitempty"`
	// IgnoreTags is a list of tags that must be ignored when determining the
	// newest version of an image. No regular expressions or glob patterns are
	// supported yet. This field is optional.
	//
	//+kubebuilder:validation:Optional
	IgnoreTags []string `json:"ignoreTags,omitempty"`
	// Platform is a string of the form <os>/<arch> that limits the tags that can
	// be considered when searching for new versions of an image. This field is
	// optional. When left unspecified, it is implicitly equivalent to the
	// OS/architecture of the Kargo controller. Care should be taken to set this
	// value correctly in cases where the image referenced by this
	// ImageRepositorySubscription will run on a Kubernetes node with a different
	// OS/architecture than the Kargo controller. At present this is uncommon, but
	// not unheard of.
	//
	//+kubebuilder:validation:Optional
	Platform string `json:"platform,omitempty"`
}

// ChartSubscription defines a subscription to a Helm chart repository.
type ChartSubscription struct {
	// RegistryURL specifies the URL of a Helm chart registry. It may be a classic
	// chart registry (using HTTP/S) OR an OCI registry. This field is required.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(((https?)|(oci))://)([\w\d\.\-]+)(:[\d]+)?(/.*)*$`
	RegistryURL string `json:"registryURL"`
	// Name specifies a Helm chart to subscribe to within the Helm chart registry
	// specified by the RegistryURL field. This field is required.
	//
	//+kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// SemverConstraint specifies constraints on what new chart versions are
	// permissible. This field is optional. When left unspecified, there will be
	// no constraints, which means the latest version of the chart will always be
	// used. Care should be taken with leaving this field unspecified, as it can
	// lead to the unanticipated rollout of breaking changes.
	//
	//+kubebuilder:validation:Optional
	SemverConstraint string `json:"semverConstraint,omitempty"`
}

// WarehouseStatus describes a Warehouse's most recently observed state.
type WarehouseStatus struct {
	// Error describes any errors that are preventing the Warehouse controller
	// from polling repositories to discover new Freight.
	Error string `json:"error,omitempty"`
	// ObservedGeneration represents the .metadata.generation that this Warehouse
	// was reconciled against.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true

// WarehouseList is a list of Warehouse resources.
type WarehouseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Warehouse `json:"items"`
}
