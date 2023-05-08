package v1alpha1

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum={SemVer,Latest,Name,Digest}
type ImageUpdateStrategy string

const (
	ImageUpdateStrategySemVer ImageUpdateStrategy = "SemVer"
	ImageUpdateStrategyLatest ImageUpdateStrategy = "Latest"
	ImageUpdateStrategyName   ImageUpdateStrategy = "Name"
	ImageUpdateStrategyDigest ImageUpdateStrategy = "Digest"
)

// +kubebuilder:validation:Enum={Image,Tag}
type ImageUpdateValueType string

const (
	ImageUpdateValueTypeImage ImageUpdateValueType = "Image"
	ImageUpdateValueTypeTag   ImageUpdateValueType = "Tag"
)

type HealthState string

const (
	HealthStateHealthy   HealthState = "Healthy"
	HealthStateUnhealthy HealthState = "Unhealthy"
	HealthStateUnknown   HealthState = "Unknown"
)

//+kubebuilder:resource:shortName={env,envs}
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name=Current State,type=string,JSONPath=`.status.currentState.id`
//+kubebuilder:printcolumn:name=Health,type=string,JSONPath=`.status.currentState.health.status`
//+kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Environment is the Kargo API's main type.
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes the sources of material used by the Environment and how
	// to incorporate newly observed materials into the Environment.
	//
	//+kubebuilder:validation:Required
	Spec *EnvironmentSpec `json:"spec"`
	// Status describes the most recently observed versions of this Environment's
	// sources of material as well as the environment's current and recent states.
	Status EnvironmentStatus `json:"status,omitempty"`
}

// EnvironmentSpec describes the sources of material used by an Environment and
// how to incorporate newly observed materials into the Environment.
type EnvironmentSpec struct {
	// Subscriptions describes the Environment's sources of material. This is a
	// required field.
	//
	//+kubebuilder:validation:Required
	Subscriptions *Subscriptions `json:"subscriptions"`
	// PromotionMechanisms describes how to incorporate newly observed materials
	// into the Environment. This is a required field.
	//
	//+kubebuilder:validation:Required
	PromotionMechanisms *PromotionMechanisms `json:"promotionMechanisms"`
	// HealthChecks describes how the health of the Environment can be assessed on
	// an ongoing basis. This is a required field.
	HealthChecks *HealthChecks `json:"healthChecks,omitempty"`
}

// Subscriptions describes an Environment's sources of material.
type Subscriptions struct {
	// Repos describes various sorts of repositories an Environment uses as
	// sources of material. This field is mutually exclusive with the UpstreamEnvs
	// field.
	Repos *RepoSubscriptions `json:"repos,omitempty"`
	// UpstreamEnvs identifies other environments as potential sources of material
	// for the Environment. This field is mutually exclusive with the Repos field.
	UpstreamEnvs []EnvironmentSubscription `json:"upstreamEnvs,omitempty"`
}

// RepoSubscriptions describes various sorts of repositories an Environment uses
// as sources of material.
type RepoSubscriptions struct {
	// Git describes subscriptions to Git repositories.
	Git []GitSubscription `json:"git,omitempty"`
	// Images describes subscriptions to container image repositories.
	Images []ImageSubscription `json:"images,omitempty"`
	// Charts describes subscriptions to Helm charts.
	Charts []ChartSubscription `json:"charts,omitempty"`
}

// GitSubscription defines a subscription to a Git repository.
type GitSubscription struct {
	// URL is the repository's URL. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^((https?://)|([\w-]+@))([\w\d\.]+)(:[\d]+)?/(.*)$`
	RepoURL string `json:"repoURL"`
	// Branch references a particular branch of the repository. This is a required
	// field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^\w+([-/]\w+)*$`
	Branch string `json:"branch"`
}

// ImageSubscription defines a subscription to an image repository.
type ImageSubscription struct {
	// RepoURL specifies the URL of the image repository to subscribe to. The
	// value in this field MUST NOT include an image tag. This field is required.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(([\w\d\.]+)(:[\d]+)?/)?[a-z0-9]+(/[a-z0-9]+)*$`
	RepoURL string `json:"repoURL"`
	// UpdateStrategy specifies the rules for how to identify the newest version
	// of the image specified by the RepoURL field. This field is optional. When
	// left unspecified, the field is implicitly treated as if its value were
	// "SemVer".
	//
	// +kubebuilder:default=SemVer
	UpdateStrategy ImageUpdateStrategy `json:"updateStrategy,omitempty"`
	// SemverConstraint specifies constraints on what new image versions are
	// permissible. This value in this field only has any effect when the
	// UpdateStrategy is SemVer or left unspecified (which is implicitly the same
	// as SemVer). This field is also optional. When left unspecified, (and the
	// UpdateStrategy is SemVer or unspecified), there will be no constraints,
	// which means the latest semantically tagged version of an image will always
	// be used. Care should be taken with leaving this field unspecified, as it
	// can lead to the unanticipated rollout of breaking changes. Refer to Image
	// Updater documentation for more details.
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
	//+kubebuilder:validation:Pattern=`^(((https?)|(oci))://)([\w\d\.]+)(:[\d]+)?(/.*)*$`
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

// EnvironmentSubscription defines a subscription to states from another
// Environment.
type EnvironmentSubscription struct {
	// Name specifies the name of an Environment.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name"`
	// Namespace specifies the namespace of the Environment. If left unspecified,
	// the namespace of the upstream repository will be defaulted to that of this
	// Environment.
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Namespace string `json:"namespace,omitempty"`
}

// PromotionMechanisms describes how to incorporate newly observed materials
// into an Environment.
type PromotionMechanisms struct {
	// GitRepoUpdates describes updates that should be applied to Git repositories
	// to incorporate newly observed materials into the Environment. This field is
	// optional, as such actions are not required in all cases.
	GitRepoUpdates []GitRepoUpdate `json:"gitRepoUpdates,omitempty"`
	// ArgoCDAppUpdates describes updates that should be applied to Argo CD
	// Application resources to incorporate newly observed materials into the
	// Environment. This field is optional, as such actions are not required in
	// all cases. Note that all updates specified by the GitRepoUpdates field, if
	// any, are applied BEFORE these.
	ArgoCDAppUpdates []ArgoCDAppUpdate `json:"argoCDAppUpdates,omitempty"`
}

// GitRepoUpdate describes updates that should be applied to a Git repository
// (using various configuration management tools) to incorporate newly observed
// materials into an Environment.
type GitRepoUpdate struct {
	// RepoURL is the URL of the repository to update. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^((https?://)|([\w-]+@))([\w\d\.]+)(:[\d]+)?/(.*)$`
	RepoURL string `json:"repoURL"`
	// ReadBranch specifies a particular branch of the repository from which to
	// locate contents that will be written to the branch specified by the
	// WriteBranch field. This field is optional. In cases where an
	// EnvironmentState includes a GitCommit, that commit's ID will supersede the
	// value of this field. Therefore, in practice, this field is only used to
	// clarify what branch of a repository can be treated as a source of manifests
	// or other configuration when an Environment has no subscription to that
	// repository.
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=`^(\w+([-/]\w+)*)?$`
	ReadBranch string `json:"readBranch"`
	// WriteBranch specifies the particular branch of the repository to be
	// updated. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^\w+([-/]\w+)*$`
	WriteBranch string `json:"writeBranch"`
	// Bookkeeper describes how to use Bookkeeper to incorporate newly observed
	// materials into the Environment. This is mutually exclusive with the
	// Kustomize and Helm fields.
	Bookkeeper *BookkeeperPromotionMechanism `json:"bookkeeper,omitempty"`
	// Kustomize describes how to use Kustomize to incorporate newly observed
	// materials into the Environment. This is mutually exclusive with the
	// Bookkeeper and Helm fields.
	Kustomize *KustomizePromotionMechanism `json:"kustomize,omitempty"`
	// Helm describes how to use Helm to incorporate newly observed materials into
	// the Environment. This is mutually exclusive with the Bookkeeper and
	// Kustomize fields.
	Helm *HelmPromotionMechanism `json:"helm,omitempty"`
}

// BookkeeperPromotionMechanism describes how to use Bookkeeper to incorporate
// newly observed materials into an Environment.
type BookkeeperPromotionMechanism struct{}

// KustomizePromotionMechanism describes how to use Kustomize to incorporate
// newly observed materials into an Environment.
type KustomizePromotionMechanism struct {
	// Images describes images for which `kustomize edit set image` should be
	// executed and the paths in which those commands should be executed.
	//
	//+kubebuilder:validation:MinItems=1
	Images []KustomizeImageUpdate `json:"images"`
}

// KustomizeImageUpdate describes how to run `kustomize edit set image`
// for a given image.
type KustomizeImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	// Path specifies a path in which the `kustomize edit set image` command
	// should be executed. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	Path string `json:"path"`
}

// HelmPromotionMechanism describes how to use Helm to incorporate newly
// observed materials into an Environment.
type HelmPromotionMechanism struct {
	// Images describes how specific image versions can be incorporated into Helm
	// values files.
	Images []HelmImageUpdate `json:"images,omitempty"`
	// Charts describes how specific chart versions can be incorporated into an
	// umbrella chart.
	Charts []HelmChartDependencyUpdate `json:"charts,omitempty"`
}

// HelmImageUpdate describes how a specific image version can be incorporated
// into a specific Helm values file.
type HelmImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(([\w\d\.]+)(:[\d]+)?/)?[a-z0-9]+(/[a-z0-9]+)*$`
	Image string `json:"image"`
	// ValuesFilePath specifies a path to the Helm values file that is to be
	// updated. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	ValuesFilePath string `json:"valuesFilePath"`
	// Key specifies a key within the Helm values file that is to be updated. This
	// is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// Value specifies the new value for the specified key in the specified Helm
	// values file. Valid values are "Image", which replaces the value of the
	// specified key with the entire <image name>:<tag>, or "Tag" which replaces
	// the value of the specified with just the new tag. This is a required field.
	Value ImageUpdateValueType `json:"value"`
}

// HelmChartDependencyUpdate describes how a specific Helm chart that is used
// as a subchart of an umbrella chart can be updated.
type HelmChartDependencyUpdate struct {
	// RegistryURL along with Name identify a subchart of the umbrella chart at
	// ChartPath whose version should be updated.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(((https?)|(oci))://)([\w\d\.]+)(:[\d]+)?(/.*)*$`
	RegistryURL string `json:"registryURL"`
	// Name along with RegistryURL identify a subchart of the umbrella chart at
	// ChartPath whose version should be updated.
	//
	//+kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// ChartPath is the path to an umbrella chart.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	ChartPath string `json:"chartPath"`
}

// ArgoCDAppUpdate describes updates that should be applied to an Argo CD
// Application resources to incorporate newly observed materials into an
// Environment.
type ArgoCDAppUpdate struct {
	// AppName specifies the name of an Argo CD Application resource to be
	// updated.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppName string `json:"appName"`
	// AppNamespace specifies the namespace of an Argo CD Application resource to
	// be updated. If left unspecified, the namespace of this Application resource
	// is defaulted to that of the Environment.
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppNamespace string `json:"appNamespace,omitempty"`
	// SourceUpdates describes updates to be applied to various sources of the
	// specified Argo CD Application resource.
	SourceUpdates []ArgoCDSourceUpdate `json:"sourceUpdates,omitempty"`
}

// ArgoCDSourceUpdate describes updates that should be applied to one of an Argo
// CD Application resource's sources.
type ArgoCDSourceUpdate struct {
	// RepoURL identifies which of the Argo CD Application's sources this update
	// is intended for. Note: As of Argo CD 2.6, Application's can use multiple
	// sources.
	//
	//+kubebuilder:validation:MinLength=1
	RepoURL string `json:"repoURL"`
	// Chart specifies a chart within a Helm chart registry if RepoURL points to a
	// Helm chart registry. Application sources that point directly at a chart do
	// so through a combination of their own RepoURL (registry) and Chart fields,
	// so BOTH of those are used as criteria in selecting an Application source to
	// update. This field MUST always be used when RepoURL points at a Helm chart
	// registry. This field MUST never be used when RepoURL points at a Git
	// repository.
	//
	//+kubebuilder:validation:Optional
	Chart string `json:"chart,omitempty"`
	// UpdateTargetRevision is a bool indicating whether the source should be
	// updated such that its TargetRevision field points at the most recently git
	// commit (if RepoURL references a git repository) or chart version (if
	// RepoURL references a chart repository).
	UpdateTargetRevision bool `json:"updateTargetRevision,omitempty"`
	// Kustomize describes updates to the source's Kustomize-specific attributes.
	Kustomize *ArgoCDKustomize `json:"kustomize,omitempty"`
	// Helm describes updates to the source's Helm-specific attributes.
	Helm *ArgoCDHelm `json:"helm,omitempty"`
}

// ArgoCDKustomize describes updates to an Argo CD Application source's
// Kustomize-specific attributes to incorporate newly observed materials into an
// Environment.
type ArgoCDKustomize struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Kustomize parameters.
	//
	//+kubebuilder:validation:MinItems=1
	Images []string `json:"images"`
}

// ArgoCDHelm describes updates to an Argo CD Application source's Helm-specific
// attributes to incorporate newly observed materials into an Environment.
type ArgoCDHelm struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Helm parameters.
	//
	//+kubebuilder:validation:MinItems=1
	Images []ArgoCDHelmImageUpdate `json:"images"`
}

// ArgoCDHelmImageUpdate describes how a specific image version can be
// incorporated into an Argo CD Application's Helm parameters.
type ArgoCDHelmImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	// Key specifies a key within an Argo CD Application's Helm parameters that is
	// to be updated. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// Value specifies the new value for the specified key in the Argo CD
	// Application's Helm parameters. Valid values are "Image", which replaces the
	// value of the specified key with the entire <image name>:<tag>, or "Tag"
	// which replaces the value of the specified with just the new tag. This is a
	// required field.
	Value ImageUpdateValueType `json:"value"`
}

// HealthChecks describes how the health of an Environment can be assessed on an
// ongoing basis.
type HealthChecks struct {
	// ArgoCDAppChecks specifies Argo CD Application resources whose sync status
	// and health should be evaluated in assessing the health of the Environment.
	ArgoCDAppChecks []ArgoCDAppCheck `json:"argoCDAppChecks,omitempty"`
}

// ArgoCDAppCheck describes a health check to perform on an Argo CD Application
// resource.
type ArgoCDAppCheck struct {
	// AppName specifies the name of the Argo CD Application resource.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppName string `json:"appName"`
	// AppNamespace specifies the namespace of the Argo CD Application resource.
	// If left unspecified, the namespace of this Application resource is
	// defaulted to be the same as that of the Environment.
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppNamespace string `json:"appNamespace,omitempty"`
}

// EnvironmentStatus describes the most recently observed versions of an
// Environment's sources of material as well as its current and recent states.
type EnvironmentStatus struct {
	// AvailableStates is a stack of available Environment states, where each
	// state is essentially a "bill of materials" describing what can be
	// automatically or manually deployed to the Environment.
	AvailableStates EnvironmentStateStack `json:"availableStates,omitempty"`
	// CurrentState is the Environment's current state -- a "bill of materials"
	// describing what is currently deployed to the Environment.
	CurrentState *EnvironmentState `json:"currentState,omitempty"`
	// History is a stack of recent Environment states, where each state is
	// essentially a "bill of materials" describing what was deployed to the
	// Environment. By default, the last ten states are stored.
	History EnvironmentStateStack `json:"history,omitempty"`
	// Error describes any errors that are preventing the Environment controller
	// from assessing Environment health, polling repositories or upstream
	// environments to discover new states, or promoting the environment to a new
	// state.
	Error string `json:"error,omitempty"`
}

// EnvironmentState is a "bill of materials" describing what was deployed to an
// Environment.
type EnvironmentState struct {
	// ID is a unique, system-assigned identifier for this state.
	ID string `json:"id,omitempty"`
	// FirstSeen represents the date/time when this EnvironmentState first entered
	// the system. This is useful and important information because it enables the
	// controller to block auto-promotion of EnvironmentStates that are older than
	// an Environment's current state, which is a case that can arise if an
	// Environment has ROLLED BACK to an older state whilst a downstream
	// Environment is already on to a newer state.
	FirstSeen *metav1.Time `json:"firstSeen,omitempty"`
	// Provenance describes the proximate source of this EnvironmentState. i.e.
	// Did it come directly from upstream repositories? Or an upstream
	// environment.
	Provenance string `json:"provenance,omitempty"`
	// Commits describes specific Git repository commits that were used in this
	// state.
	Commits []GitCommit `json:"commits,omitempty"`
	// Images describes container images and versions thereof that were used
	// in this state.
	Images []Image `json:"images,omitempty"`
	// Charts describes Helm charts that were used in this state.
	Charts []Chart `json:"charts,omitempty"`
	// Health is the state's last observed health. If this state is the
	// Environment's current state, this will be continuously re-assessed and
	// updated. If this state is a past state of the Environment, this field will
	// denote the last observed health state before transitioning into a different
	// state.
	Health *Health `json:"health,omitempty"`
}

func (e *EnvironmentState) UpdateStateID() {
	materials := []string{}
	for _, commit := range e.Commits {
		materials = append(
			materials,
			fmt.Sprintf("%s:%s", commit.RepoURL, commit.ID),
		)
	}
	for _, image := range e.Images {
		materials = append(
			materials,
			fmt.Sprintf("%s:%s", image.RepoURL, image.Tag),
		)
	}
	for _, chart := range e.Charts {
		materials = append(
			materials,
			fmt.Sprintf("%s/%s:%s", chart.RegistryURL, chart.Name, chart.Version),
		)
	}
	sort.Strings(materials)
	e.ID = fmt.Sprintf(
		"%x",
		sha1.Sum([]byte(strings.Join(materials, "|"))),
	)
}

type EnvironmentStateStack []EnvironmentState

// Empty returns a bool indicating whether or not the EnvironmentStateStack is
// empty. nil counts as empty.
func (e EnvironmentStateStack) Empty() bool {
	return len(e) == 0
}

// Pop removes and returns the leading element from EnvironmentStateStack. If
// the EnvironmentStateStack is empty, the EnvironmentStack is not modified and
// a empty EnvironmentState is returned instead. A boolean is also returned
// indicating whether the returned EnvironmentState came from the top of the
// stack (true) or is a zero value for that type (false).
func (e *EnvironmentStateStack) Pop() (EnvironmentState, bool) {
	item, ok := e.Top()
	if ok {
		*e = (*e)[1:]
	}
	return item, ok
}

// Top returns the leading element from EnvironmentStateStack without modifying
// the EnvironmentStateStack. If the EnvironmentStateStack is empty, an empty
// EnvironmentState is returned instead. A boolean is also returned indicating
// whether the returned EnvironmentState came from the top of the stack (true)
// or is a zero value for that type (false).
func (e EnvironmentStateStack) Top() (EnvironmentState, bool) {
	if e.Empty() {
		return EnvironmentState{}, false
	}
	item := *e[0].DeepCopy()
	return item, true
}

// Push pushes one or more EnvironmentStates onto the EnvironmentStateStack. The
// order of the new elements at the top of the stack will be equal to the order
// in which they were passed to this function. i.e. The first new element passed
// will be the element at the top of the stack. If resulting modification grow
// the depth of the stack beyond 10 elements, the stack is truncated at the
// bottom. i.e. Modified to contain only the top 10 elements.
func (e *EnvironmentStateStack) Push(states ...EnvironmentState) {
	*e = append(states, *e...)
	const max = 10
	if len(*e) > max {
		*e = (*e)[:max]
	}
}

// Image describes a specific version of a container image.
type Image struct {
	// RepoURL describes the repository in which the image can be found.
	RepoURL string `json:"repoURL,omitempty"`
	// Tag identifies a specific version of the image in the repository specified
	// by RepoURL.
	Tag string `json:"tag,omitempty"`
}

// Chart describes a specific version of a Helm chart.
type Chart struct {
	// RepoURL specifies the remote registry in which this chart is located.
	RegistryURL string `json:"registryURL,omitempty"`
	// Name specifies the name of the chart.
	Name string `json:"name,omitempty"`
	// Version specifies a particular version of the chart.
	Version string `json:"version,omitempty"`
}

// GitCommit describes a specific commit from a specific Git repository.
type GitCommit struct {
	// RepoURL is the URL of a Git repository.
	RepoURL string `json:"repoURL,omitempty"`
	// ID is the ID of a specific commit in the Git repository specified by
	// RepoURL.
	ID string `json:"id,omitempty"`
	// Branch denotes the branch of the repository where this commit was found.
	Branch string `json:"branch,omitempty"`
	// HealthCheckCommit is the ID of a specific commit. When specified,
	// assessments of Environment health will used this value (instead of ID) when
	// determining if applicable sources of Argo CD Application resources
	// associated with the environment are or are not synced to this commit. Note
	// that there are cases (as in that of Bookkeeper being utilized as a
	// promotion mechanism) wherein the value of this field may differ from the
	// commit ID found in the ID field.
	HealthCheckCommit string `json:"healthCheckCommit,omitempty"`
}

// Equals returns a bool indicating whether two GitCommits are equivalent.
func (g *GitCommit) Equals(rhs *GitCommit) bool {
	if g == nil && rhs == nil {
		return true
	}
	if (g == nil && rhs != nil) || (g != nil && rhs == nil) {
		return false
	}
	// If we get to here, both operands are non-nil
	return g.RepoURL == rhs.RepoURL && g.ID == rhs.ID
}

// Health describes the health of an Environment.
type Health struct {
	// Status describes the health of the Environment.
	Status HealthState `json:"status,omitempty"`
	// StatusReason clarifies why an Environment in any state other than Healthy
	// is in that state. The value of this field will always be the empty string
	// when an Environment is Healthy.
	StatusReason string `json:"statusReason,omitempty"`
}

//+kubebuilder:object:root=true

// EnvironmentList is a list of Environment resources.
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
}
