package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:validation:Enum={Lexical,NewestFromBranch,NewestTag,SemVer}
type CommitSelectionStrategy string

const (
	CommitSelectionStrategyLexical          CommitSelectionStrategy = "Lexical"
	CommitSelectionStrategyNewestFromBranch CommitSelectionStrategy = "NewestFromBranch"
	CommitSelectionStrategyNewestTag        CommitSelectionStrategy = "NewestTag"
	CommitSelectionStrategySemVer           CommitSelectionStrategy = "SemVer"
)

// +kubebuilder:validation:Enum={Digest,Lexical,NewestBuild,SemVer}
type ImageSelectionStrategy string

const (
	ImageSelectionStrategyDigest      ImageSelectionStrategy = "Digest"
	ImageSelectionStrategyLexical     ImageSelectionStrategy = "Lexical"
	ImageSelectionStrategyNewestBuild ImageSelectionStrategy = "NewestBuild"
	ImageSelectionStrategySemVer      ImageSelectionStrategy = "SemVer"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name=Shard,type=string,JSONPath=`.spec.shard`
//+kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Warehouse is a source of Freight.
type Warehouse struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes sources of artifacts.
	//
	//+kubebuilder:validation:Required
	Spec *WarehouseSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the Warehouse's most recently observed state.
	Status WarehouseStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

func (w *Warehouse) GetStatus() *WarehouseStatus {
	return &w.Status
}

// WarehouseSpec describes sources of versioned artifacts to be included in
// Freight produced by this Warehouse.
type WarehouseSpec struct {
	// Shard is the name of the shard that this Warehouse belongs to. This is an
	// optional field. If not specified, the Warehouse will belong to the default
	// shard. A defaulting webhook will sync this field with the value of the
	// kargo.akuity.io/shard label. When the shard label is not present or differs
	// from the value of this field, the defaulting webhook will set the label to
	// the value of this field. If the shard label is present and this field is
	// empty, the defaulting webhook will set the value of this field to the value
	// of the shard label.
	Shard string `json:"shard,omitempty" protobuf:"bytes,2,opt,name=shard"`
	// Subscriptions describes sources of artifacts to be included in Freight
	// produced by this Warehouse.
	//
	//+kubebuilder:validation:MinItems=1
	Subscriptions []RepoSubscription `json:"subscriptions" protobuf:"bytes,1,rep,name=subscriptions"`
}

// RepoSubscription describes a subscription to ONE OF a Git repository, a
// container image repository, or a Helm chart repository.
type RepoSubscription struct {
	// Git describes a subscriptions to a Git repository.
	Git *GitSubscription `json:"git,omitempty" protobuf:"bytes,1,opt,name=git"`
	// Image describes a subscription to container image repository.
	Image *ImageSubscription `json:"image,omitempty" protobuf:"bytes,2,opt,name=image"`
	// Chart describes a subscription to a Helm chart repository.
	Chart *ChartSubscription `json:"chart,omitempty" protobuf:"bytes,3,opt,name=chart"`
}

// GitSubscription defines a subscription to a Git repository.
type GitSubscription struct {
	// URL is the repository's URL. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^https?://(\w+([\.-]\w+)*@)?\w+([\.-]\w+)*(:[\d]+)?(/.*)?$`
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// CommitSelectionStrategy specifies the rules for how to identify the newest
	// commit of interest in the repository specified by the RepoURL field. This
	// field is optional. When left unspecified, the field is implicitly treated
	// as if its value were "NewestFromBranch".
	//
	// +kubebuilder:default=NewestFromBranch
	CommitSelectionStrategy CommitSelectionStrategy `json:"commitSelectionStrategy,omitempty" protobuf:"bytes,2,opt,name=commitSelectionStrategy"`
	// Branch references a particular branch of the repository. The value in this
	// field only has any effect when the CommitSelectionStrategy is
	// NewestFromBranch or left unspecified (which is implicitly the same as
	// NewestFromBranch). This field is optional. When left unspecified, (and the
	// CommitSelectionStrategy is NewestFromBranch or unspecified), the
	// subscription is implicitly to the repository's default branch.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^\w+([-/]\w+)*$`
	Branch string `json:"branch,omitempty" protobuf:"bytes,3,opt,name=branch"`
	// SemverConstraint specifies constraints on what new tagged commits are
	// considered in determining the newest commit of interest. The value in this
	// field only has any effect when the CommitSelectionStrategy is SemVer. This
	// field is optional. When left unspecified, there will be no constraints,
	// which means the latest semantically tagged commit will always be used. Care
	// should be taken with leaving this field unspecified, as it can lead to the
	// unanticipated rollout of breaking changes.
	//
	//+kubebuilder:validation:Optional
	SemverConstraint string `json:"semverConstraint,omitempty" protobuf:"bytes,4,opt,name=semverConstraint"`
	// AllowTags is a regular expression that can optionally be used to limit the
	// tags that are considered in determining the newest commit of interest. The
	// value in this field only has any effect when the CommitSelectionStrategy is
	// Lexical, NewestTag, or SemVer. This field is optional.
	//
	//+kubebuilder:validation:Optional
	AllowTags string `json:"allowTags,omitempty" protobuf:"bytes,5,opt,name=allowTags"`
	// IgnoreTags is a list of tags that must be ignored when determining the
	// newest commit of interest. No regular expressions or glob patterns are
	// supported yet. The value in this field only has any effect when the
	// CommitSelectionStrategy is Lexical, NewestTag, or SemVer. This field is
	// optional.
	//
	//+kubebuilder:validation:Optional
	IgnoreTags []string `json:"ignoreTags,omitempty" protobuf:"bytes,6,rep,name=ignoreTags"`
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when connecting to the repository. This should be enabled
	// only with great caution.
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
	// ScanPaths is a list of regular expressions that can optinally be used to
	// limit file paths in repository, changes in which will result in creation of
	// new freight. When not specified - changes in any path will produce new
	// freight, it is equivalent to having a ScanPaths with an entry of ".*"
	// When both ScanPaths and IgnorePaths are specified and match same path/paths,
	// IgnorePaths will prevail over ScanPaths.
	//+kubebuilder:validation:Optional
	ScanPaths []string `json:"scanPaths,omitempty"`
	// IgnorePaths is an optional list of regular expressions used to specify paths
	// in git repository, changes in which should never produce new freight. When used
	// in conjuction with ScanPaths, both matching same path/paths, IgnorePaths takes
	// precedence over ScanPaths.
	//+kubebuilder:validation:Optional
	IgnorePaths []string `json:"ignorePaths,omitempty"`
}

// ImageSubscription defines a subscription to an image repository.
type ImageSubscription struct {
	// RepoURL specifies the URL of the image repository to subscribe to. The
	// value in this field MUST NOT include an image tag. This field is required.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(\w+([\.-]\w+)*(:[\d]+)?/)?(\w+([\.-]\w+)*)(/\w+([\.-]\w+)*)*$`
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// GitRepoURL optionally specifies the URL of a Git repository that contains
	// the source code for the image repository referenced by the RepoURL field.
	// When this is specified, Kargo MAY be able to infer and link to the exact
	// revision of that source code that was used to build the image.
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=`^https?://(\w+([\.-]\w+)*@)?\w+([\.-]\w+)*(:[\d]+)?(/.*)?$`
	GitRepoURL string `json:"gitRepoURL,omitempty" protobuf:"bytes,2,opt,name=gitRepoURL"`
	// ImageSelectionStrategy specifies the rules for how to identify the newest version
	// of the image specified by the RepoURL field. This field is optional. When
	// left unspecified, the field is implicitly treated as if its value were
	// "SemVer".
	//
	// +kubebuilder:default=SemVer
	ImageSelectionStrategy ImageSelectionStrategy `json:"imageSelectionStrategy,omitempty" protobuf:"bytes,3,opt,name=imageSelectionStrategy"`
	// SemverConstraint specifies constraints on what new image versions are
	// permissible. The value in this field only has any effect when the
	// ImageSelectionStrategy is SemVer or left unspecified (which is implicitly
	// the same as SemVer). This field is also optional. When left unspecified,
	// (and the ImageSelectionStrategy is SemVer or unspecified), there will be no
	// constraints, which means the latest semantically tagged version of an image
	// will always be used. Care should be taken with leaving this field
	// unspecified, as it can lead to the unanticipated rollout of breaking
	// changes. Refer to Image Updater documentation for more details.
	// More info: https://github.com/masterminds/semver#checking-version-constraints
	//
	//+kubebuilder:validation:Optional
	SemverConstraint string `json:"semverConstraint,omitempty" protobuf:"bytes,4,opt,name=semverConstraint"`
	// AllowTags is a regular expression that can optionally be used to limit the
	// image tags that are considered in determining the newest version of an
	// image. This field is optional.
	//
	//+kubebuilder:validation:Optional
	AllowTags string `json:"allowTags,omitempty" protobuf:"bytes,5,opt,name=allowTags"`
	// IgnoreTags is a list of tags that must be ignored when determining the
	// newest version of an image. No regular expressions or glob patterns are
	// supported yet. This field is optional.
	//
	//+kubebuilder:validation:Optional
	IgnoreTags []string `json:"ignoreTags,omitempty" protobuf:"bytes,6,rep,name=ignoreTags"`
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
	Platform string `json:"platform,omitempty" protobuf:"bytes,7,opt,name=platform"`
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when connecting to the repository. This should be enabled
	// only with great caution.
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty" protobuf:"varint,8,opt,name=insecureSkipTLSVerify"`
}

// ChartSubscription defines a subscription to a Helm chart repository.
type ChartSubscription struct {
	// RepoURL specifies the URL of a Helm chart repository. It may be a classic
	// chart repository (using HTTP/S) OR a repository within an OCI registry.
	// Classic chart repositories can contain differently named charts. When this
	// field points to such a repository, the Name field MUST also be used
	// to specify the name of the desired chart within that repository. In the
	// case of a repository within an OCI registry, the URL implicitly points to
	// a specific chart and the Name field MUST NOT be used. The RepoURL field is
	// required.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(((https?)|(oci))://)([\w\d\.\-]+)(:[\d]+)?(/.*)*$`
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// Name specifies the name of a Helm chart to subscribe to within a classic
	// chart repository specified by the RepoURL field. This field is required
	// when the RepoURL field points to a classic chart repository and MUST
	// otherwise be empty.
	Name string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	// SemverConstraint specifies constraints on what new chart versions are
	// permissible. This field is optional. When left unspecified, there will be
	// no constraints, which means the latest version of the chart will always be
	// used. Care should be taken with leaving this field unspecified, as it can
	// lead to the unanticipated rollout of breaking changes.
	// More info: https://github.com/masterminds/semver#checking-version-constraints
	//
	//+kubebuilder:validation:Optional
	SemverConstraint string `json:"semverConstraint,omitempty" protobuf:"bytes,3,opt,name=semverConstraint"`
}

// WarehouseStatus describes a Warehouse's most recently observed state.
type WarehouseStatus struct {
	// Message describes any errors that are preventing the Warehouse controller
	// from polling repositories to discover new Freight.
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
	// ObservedGeneration represents the .metadata.generation that this Warehouse
	// was reconciled against.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// LastFreight refers to the last Freight produced by this Warehouse
	LastFreight *FreightReference `json:"lastFreight,omitempty"`
}

//+kubebuilder:object:root=true

// WarehouseList is a list of Warehouse resources.
type WarehouseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Warehouse `json:"items" protobuf:"bytes,2,rep,name=items"`
}
