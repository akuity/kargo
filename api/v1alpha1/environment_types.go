package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ImageUpdateStrategy string

const (
	ImageUpdateStrategySemVer ImageUpdateStrategy = "SemVer"
	ImageUpdateStrategyLatest ImageUpdateStrategy = "Latest"
	ImageUpdateStrategyName   ImageUpdateStrategy = "Name"
	ImageUpdateStrategyDigest ImageUpdateStrategy = "Digest"
)

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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Environment is the Kargo API's main type.
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes the sources of material used by the Environment and how
	// to incorporate newly observed materials into the Environment.
	Spec EnvironmentSpec `json:"spec,omitempty"`
	// Status describes the most recently observed versions of this Environment's
	// sources of material as well as the environment's current and recent states.
	Status EnvironmentStatus `json:"status,omitempty"`
}

// EnvironmentSpec describes the sources of material used by an Environment and
// how to incorporate newly observed materials into the Environment.
type EnvironmentSpec struct {
	// Subscriptions describes the Environment's sources of material. This is a
	// required field.
	Subscriptions Subscriptions `json:"subscriptions,omitempty"`
	// PromotionMechanisms describes how to incorporate newly observed materials
	// into the Environment. This is a required field.
	PromotionMechanisms PromotionMechanisms `json:"promotionMechanisms,omitempty"` // nolint: lll
	// HealthChecks describes how the health of the Environment can be assessed on
	// an ongoing basis. This is a required field.
	HealthChecks HealthChecks `json:"healthChecks,omitempty"`
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
	RepoURL string `json:"repoURL,omitempty"`
	// Branch references a particular branch of the repository. This field is
	// optional. Leaving this unspecified is equivalent to specifying the
	// repository's default branch, whatever that may happen to be -- typically
	// "main" or "master".
	Branch string `json:"branch,omitempty"`
}

// ImageSubscription defines a subscription to an image repository.
type ImageSubscription struct {
	// RepoURL specifies the URL of the image repository to subscribe to. The
	// value in this field MUST NOT include an image tag. This field is required.
	RepoURL string `json:"repoURL,omitempty"`
	// UpdateStrategy specifies the rules for how to identify the newest version
	// of the image specified by the RepoURL field. This field is optional. When
	// left unspecified, the field is implicitly treated as if its value were
	// "SemVer".
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
	SemverConstraint string `json:"semverConstraint,omitempty"`
	// AllowTags is a regular expression that can optionally be used to limit the
	// image tags that are considered in determining the newest version of an
	// image. This field is optional.
	AllowTags string `json:"allowTags,omitempty"`
	// IgnoreTags is a list of tags that must be ignored when determining the
	// newest version of an image. No regular expressions or glob patterns are
	// supported yet. This field is optional.
	IgnoreTags []string `json:"ignoreTags,omitempty"`
	// Platform is a string of the form <os>/<arch> that limits the tags that can
	// be considered when searching for new versions of an image. This field is
	// optional. When left unspecified, it is implicitly equivalent to the
	// OS/architecture of the Kargo controller. Care should be taken to set this
	// value correctly in cases where the image referenced by this
	// ImageRepositorySubscription will run on a Kubernetes node with a different
	// OS/architecture than the Kargo controller. At present this is uncommon, but
	// not unheard of.
	Platform string `json:"platform,omitempty"`
	// PullSecret is a reference to a Kubernetes Secret containing repository
	// credentials. If left unspecified, Kargo will fall back on globally
	// configured repository credentials, if they exist.
	PullSecret string `json:"pullSecret,omitempty"`
}

// ChartSubscription defines a subscription to a Helm chart repository.
type ChartSubscription struct {
	// RegistryURL specifies the URL of a Helm chart registry. It may be a classic
	// chart registry (using HTTP/S) OR an OCI registry. This field is required.
	RegistryURL string `json:"registryURL,omitempty"`
	// Name specifies a Helm chart to subscribe to within the Helm chart registry
	// specified by the RegistryURL field. This field is required.
	Name string `json:"name,omitempty"`
	// SemverConstraint specifies constraints on what new chart versions are
	// permissible. This field is optional. When left unspecified, there will be
	// no constraints, which means the latest version of the chart will always be
	// used. Care should be taken with leaving this field unspecified, as it can
	// lead to the unanticipated rollout of breaking changes.
	SemverConstraint string `json:"semverConstraint,omitempty"`
}

// EnvironmentSubscription defines a subscription to states from another
// Environment.
type EnvironmentSubscription struct {
	// Name specifies the name of an Environment.
	Name string `json:"name,omitempty"`
	// Namespace specifies the namespace of the Environment.
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
	RepoURL string `json:"repoURL,omitempty"`
	// Branch references a particular branch of the repository to be updated. This
	// field is optional. Leaving this unspecified is equivalent to specifying the
	// repository's default branch, whatever that may happen to be -- typically
	// "main" or "master".
	Branch string `json:"branch,omitempty"`
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
	Images []KustomizeImageUpdate `json:"images,omitempty"`
}

// KustomizeImageUpdate describes how to run `kustomize edit set image`
// for a given image.
type KustomizeImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	Image string `json:"image,omitempty"`
	// Path specifies a path in which the `kustomize edit set image` command
	// should be executed. This is a required field.
	Path string `json:"path,omitempty"`
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
	Image string `json:"image,omitempty"`
	// ValuesFilePath specifies a path to the Helm values file that is to be
	// updated. This is a required field.
	ValuesFilePath string `json:"valuesFilePath,omitempty"`
	// Key specifies a key within the Helm values file that is to be updated. This
	// is a required field.
	Key string `json:"key,omitempty"`
	// Value specifies the new value for the specified key in the specified Helm
	// values file. Valid values are "Image", which replaces the value of the
	// specified key with the entire <image name>:<tag>, or "Tag" which replaces
	// the value of the specified with just the new tag. This is a required field.
	Value ImageUpdateValueType `json:"value,omitempty"`
}

// HelmChartDependencyUpdate describes how a specific Helm chart that is used
// as a subchart of an umbrella chart can be updated.
type HelmChartDependencyUpdate struct {
	// RegistryURL along with Name identify a subchart of the umbrella chart at
	// ChartPath whose version should be updated.
	RegistryURL string `json:"registryURL,omitempty"`
	// Name along with RegistryURL identify a subchart of the umbrella chart at
	// ChartPath whose version should be updated.
	Name string `json:"name,omitempty"`
	// ChartPath is the path to an umbrella chart.
	ChartPath string `json:"chartPath,omitempty"`
}

// ArgoCDAppUpdate describes updates that should be applied to an Argo CD
// Application resources to incorporate newly observed materials into an
// Environment.
type ArgoCDAppUpdate struct {
	// AppName specifies the name of an Argo CD Application resource to be
	// updated.
	AppName string `json:"appName,omitempty"`
	// AppNamespace specifies the namespace of an Argo CD Application resource to
	// be updated.
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
	RepoURL string `json:"repoURL,omitempty"`
	// Chart specifies a chart within a Helm chart registry if RepoURL points to a
	// Helm chart registry. Application sources that point directly at a chart do
	// so through a combination of their own RepoURL (registry) and Chart fields,
	// so BOTH of those are used as criteria in selecting an Application source to
	// update. This field MUST always be used when RepoURL points at a Helm chart
	// registry. This field MUST never be used when RepoURL points at a Git
	// repository.
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
	Images []string `json:"images,omitempty"`
}

// ArgoCDHelm describes updates to an Argo CD Application source's Helm-specific
// attributes to incorporate newly observed materials into an Environment.
type ArgoCDHelm struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Helm parameters.
	Images []ArgoCDHelmImageUpdate `json:"images,omitempty"`
}

// ArgoCDHelmImageUpdate describes how a specific image version can be
// incorporated into an Argo CD Application's Helm parameters.
type ArgoCDHelmImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	Image string `json:"image,omitempty"`
	// Key specifies a key within an Argo CD Application's Helm parameters that is
	// to be updated. This is a required field.
	Key string `json:"key,omitempty"`
	// Value specifies the new value for the specified key in the Argo CD
	// Application's Helm parameters. Valid values are "Image", which replaces the
	// value of the specified key with the entire <image name>:<tag>, or "Tag"
	// which replaces the value of the specified with just the new tag. This is a
	// required field.
	Value ImageUpdateValueType `json:"value,omitempty"`
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
	AppName string `json:"appName,omitempty"`
	// AppNamespace specifies the namespace of the Argo CD Application resource.
	AppNamespace string `json:"appNamespace,omitempty"`
}

// EnvironmentStatus describes the most recently observed versions of an
// Environment's sources of material as well as its current and recent states.
type EnvironmentStatus struct {
	// AvailableStates is a stack of available Environment states, where each
	// state is essentially a "bill of materials" describing what can be
	// automatically or manually deployed to the Environment.
	AvailableStates EnvironmentStateStack `json:"availableStates,omitempty"`
	// States is a stack of recent Environment states, where each state is
	// essentially a "bill of materials" describing what was deployed to the
	// Environment. By default, the last ten states are stored.
	States EnvironmentStateStack `json:"states,omitempty"`
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

type EnvironmentStateStack []EnvironmentState

// Empty returns a bool indicating whether or not the EnvironmentStateStack is
// empty. nil counts as empty.
func (e EnvironmentStateStack) Empty() bool {
	return len(e) == 0
}

// Pop returns the EnvironmentStateStack with its leading element removed as
// well as the leading element itself if the EnvironmentStateStack is not empty.
// nil counts as empty. When the EnvironmentStateStack is empty, the
// EnvironmentStack is returned unmodified with a new EnvironmentStateStack. A
// boolean is also returned indicating whether the returned
// EnvironmentStateStack came from the top of the stack (true) or is a zero
// value for that type (false).
func (e EnvironmentStateStack) Pop() (
	EnvironmentStateStack,
	EnvironmentState,
	bool,
) {
	if e.Empty() {
		return e, EnvironmentState{}, false
	}
	return e[1:], *e[0].DeepCopy(), true
}

// Push pushes one or more EnvironmentStates onto the EnvironmentStateStack. The
// order of the new elements at the top of the stack will be equal to the order
// in which they were passed to this function. i.e. The first new element passed
// will be the element at the top of the stack. If resulting modification grow
// the depth of the stack beyond 10 elements, the stack is truncated at the
// bottom. i.e. Modified to contain only the top 10 elements. In all cases, the
// modified stack is returned.
func (e EnvironmentStateStack) Push(
	states ...EnvironmentState,
) EnvironmentStateStack {
	e = append(states, e...)
	const max = 10
	if len(e) > max {
		return e[:max]
	}
	return e
}

// SameMaterials returns a bool indicating whether or not two EnvironmentStates
// are composed of the same materials.
func (e *EnvironmentState) SameMaterials(rhs *EnvironmentState) bool {
	if e == nil && rhs == nil {
		return true
	}
	if (e == nil && rhs != nil) || (e != nil && rhs == nil) {
		return false
	}
	if len(e.Commits) != len(rhs.Commits) {
		return false
	}
	if len(e.Images) != len(rhs.Images) {
		return false
	}
	if len(e.Charts) != len(rhs.Charts) {
		return false
	}

	// The order of commits shouldn't matter. We have some work to do to make an
	// effective comparison...
	lhsCommits := map[string]struct{}{}
	for _, lCommit := range e.Commits {
		lhsCommits[fmt.Sprintf("%s:%s", lCommit.RepoURL, lCommit.ID)] = struct{}{}
	}
	for _, rCommit := range rhs.Commits {
		if _, exists :=
			lhsCommits[fmt.Sprintf("%s:%s", rCommit.RepoURL, rCommit.ID)]; !exists {
			return false
		}
	}

	// The order of images shouldn't matter. We have some work to do to make an
	// effective comparison...
	lhsImgVersions := map[string]struct{}{}
	for _, lImg := range e.Images {
		lhsImgVersions[fmt.Sprintf("%s:%s", lImg.RepoURL, lImg.Tag)] = struct{}{}
	}
	for _, rImg := range rhs.Images {
		if _, exists :=
			lhsImgVersions[fmt.Sprintf("%s:%s", rImg.RepoURL, rImg.Tag)]; !exists {
			return false
		}
	}

	// The order of charts shouldn't matter. We have some work to do to make an
	// effective comparison...
	lhsChartVersions := map[string]struct{}{}
	for _, lChart := range e.Charts {
		lhsChartVersions[fmt.Sprintf("%s:%s:%s", lChart.RegistryURL, lChart.Name, lChart.Version)] = struct{}{} // nolint: lll
	}
	for _, rChart := range rhs.Charts {
		if _, exists := lhsChartVersions[fmt.Sprintf("%s:%s:%s", rChart.RegistryURL, rChart.Name, rChart.Version)]; !exists { // nolint: lll
			return false
		}
	}
	return true
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
