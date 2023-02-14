package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// GitRepo encapsulates the details of a Git repository. This field is a
	// single place for these details, which are used or referenced widely
	// throughout this API.
	GitRepo *GitRepo `json:"gitRepo,omitempty"`
	// Subscriptions describes the Environment's sources of material. This is a
	// required field.
	Subscriptions *Subscriptions `json:"subscriptions,omitempty"`
	// PromotionMechanisms describes how to incorporate newly observed materials
	// into the Environment. This is a required field.
	PromotionMechanisms *PromotionMechanisms `json:"promotionMechanisms,omitempty"` // nolint: lll
	// HealthChecks describes how the health of the Environment can be assessed on
	// an ongoing basis. This is a required field.
	HealthChecks *HealthChecks `json:"healthChecks,omitempty"`
}

// GitRepo encapsulates the details of a Git repository.
type GitRepo struct {
	// URL is the repository's URL. This is a required field.
	URL string `json:"url,omitempty"`
	// Branch references a particular branch of the repository. This field is
	// optional. Leaving this unspecified is equivalent to specifying the
	// repository's default branch, whatever that may happen to be -- typically
	// "main" or "master".
	Branch string `json:"branch,omitempty"`
}

// Subscriptions describes an Environment's sources of material.
type Subscriptions struct {
	// Repos describes various sorts of repositories an Environment uses as
	// sources of material. This field is mutually exclusive with the UpstreamEnvs
	// field.
	Repos *RepoSubscriptions `json:"repos,omitempty"`
	// UpstreamEnvs identifies other environments as potential sources of material
	// for the Environment. This field is mutually exclusive with the Repos field.
	UpstreamEnvs []string `json:"upstreamEnvs,omitempty"`
}

// RepoSubscriptions describes various sorts of repositories an Environment uses
// as sources of material.
type RepoSubscriptions struct {
	// Git indicates, when true, that there is a subscription to the Git
	// repository described by the Environment's GitRepo field. When false, there
	// is no such subscription.
	Git bool `json:"git,omitempty"`
	// Images describes subscriptions to container image repositories.
	Images []ImageSubscription `json:"images,omitempty"`
	// Charts describes subscriptions to Helm charts.
	Charts []ChartSubscription `json:"charts,omitempty"`
}

// ImageSubscription defines a subscription to an image repository.
type ImageSubscription struct {
	// RepoURL specifies the URL of the image repository to subscribe to. The
	// value in this field MUST NOT include an image tag. This field is required.
	RepoURL string `json:"repoURL,omitempty"`

	// TODO: Make UpdateStrategy its own type

	// UpdateStrategy specifies the rules for how to identify the newest version
	// of the image specified by the RepoURL field. This field is optional. When
	// left unspecified, the field is implicitly treated as if its value were
	// "SemVer".
	UpdateStrategy string `json:"updateStrategy,omitempty"`
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

// PromotionMechanisms describes how incorporate newly observed materials into
// an Environment.
type PromotionMechanisms struct {
	// Git describes actions that should be applied to a Git repository to
	// incorporate newly observed materials into the Environment. This field is
	// optional, as such actions are not required in all cases.
	Git *GitPromotionMechanism `json:"git,omitempty"` // nolint: lll
	// ArgoCD describes actions that should be taken in Argo CD to incorporate
	// newly observed materials into the Environment. This field is optional, as
	// such actions are not required in all cases. Note that all actions specified
	// by the ConfigManagement field, if any, are applied BEFORE these actions.
	ArgoCD *ArgoCDPromotionMechanism `json:"argoCD,omitempty"`
}

// GitPromotionMechanism describes actions that should be applied to a Git
// repository (using various configuration management tools) to incorporate
// newly observed materials into an Environment.
type GitPromotionMechanism struct {
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
type BookkeeperPromotionMechanism struct {
	// TODO: Document this
	TargetBranch string `json:"targetBranch,omitempty"`
}

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
	// TODO: Document this
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

// TODO: Document this
type HelmChartDependencyUpdate struct {
	// TODO: Document this
	RegistryURL string `json:"registryURL,omitempty"`
	// TODO: Document this
	Name string `json:"name,omitempty"`
	// TODO: Document this
	ChartPath string `json:"chartPath,omitempty"`
}

// ArgoCDPromotionMechanism describes actions that should be taken in Argo CD to
// incorporate newly observed materials into an Environment.
type ArgoCDPromotionMechanism struct {
	// AppUpdates describes updates that should be applied to various Argo CD
	// Application resources to incorporate newly observed materials into the
	// Environment.
	AppUpdates []ArgoCDAppUpdate `json:"appUpdates,omitempty"`
}

// ArgoCDAppUpdate describes updates that should be applied to various Argo CD
// Application resources to incorporate newly observed materials into an
// Environment.
type ArgoCDAppUpdate struct {
	// Name specifies the name of an Argo CD Application resource to be updated.
	Name string `json:"name,omitempty"`
	// RefreshAndSync is a bool indicating whether the specified Argo CD
	// Application resource should be forcefully synced and refreshed. It defaults
	// to false. You should set this to true if your Argo CD Application should
	// sync and refresh after other promotion mechanisms (e.g. config
	// management-based mechanisms) have been applied and your Argo CD Application
	// is configured NOT to automatically sync and refresh. You may also set this
	// to true even if your Argo CD Application IS configured to automatically
	// sync and refresh and you would simply like to accelerate the promotion
	// process.
	RefreshAndSync bool `json:"refreshAndSync,omitempty"`
	// UpdateTargetRevision is a bool indicating whether the specified Argo CD
	// Application resource should be updated such that its TargetRevision field
	// points at the most recently observed commit in the Environment's Git
	// repository. It defaults to false. When set to true, the affected
	// Application resource will also be forcefully synced and refreshed after
	// such an update regardless of the value of the RefreshAndSync field.
	UpdateTargetRevision bool `json:"updateTargetRevision,omitempty"`
	// Kustomize describes updates to an Argo CD Application's Kustomize-specific
	// attributes. The affected Application resource will also be forcefully
	// synced and refreshed after such an update regardless of the value of the
	// RefreshAndSync field.
	Kustomize *ArgoCDKustomize `json:"kustomize,omitempty"`
	// Helm describes updates to an Argo CD Application's Helm-specific
	// attributes. The affected Application resource will also be forcefully
	// synced and refreshed after such an update regardless of the value of the
	// RefreshAndSync field.
	Helm *ArgoCDHelm `json:"helm,omitempty"`
}

// ArgoCDKustomize describes updates to an Argo CD Application's
// Kustomize-specific attributes to incorporate newly observed materials into an
// Environment.
type ArgoCDKustomize struct {
	// TODO: Document this
	Images []string `json:"images,omitempty"`
}

// ArgoCDHelm describes updates to an Argo CD Application's Helm-specific
// attributes to incorporate newly observed materials into an Environment.
type ArgoCDHelm struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Helm parameters.
	Images []ArgoCDHelmImageUpdate `json:"images,omitempty"`
	// TODO: Document this
	Chart *ArgoCDHelmChartUpdate `json:"chart,omitempty"`
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

// ArgoCDHelmChartUpdate describes how a specific version of a Helm Chart can be
// incorporated into an Argo CD Application.
type ArgoCDHelmChartUpdate struct {
	// TODO: Document this
	RegistryURL string `json:"registryURL,omitempty"`
	// TODO: Document this
	Name string `json:"name,omitempty"`
}

// HealthChecks describes how the health of an Environment can be assessed on an
// ongoing basis.
type HealthChecks struct {
	// ArgoCDApps specifies Argo CD Application resources whose sync status and
	// health should be evaluated in assessing the health of the Environment.
	ArgoCDApps []string `json:"argoCDApps,omitempty"`
}

// EnvironmentStatus describes the most recently observed versions of an
// Environment's sources of material as well as its current and recent states.
type EnvironmentStatus struct {
	// AvailableStates is a stack of available Environment states, where each
	// state is essentially a "bill of materials" describing what can be
	// automatically or manually deployed to the Environment.
	AvailableStates []EnvironmentState `json:"availableStates,omitempty"`
	// States is a stack of recent Environment states, where each state is
	// essentially a "bill of materials" describing what was deployed to the
	// Environment. By default, the last ten states are stored.
	States []EnvironmentState `json:"states,omitempty"`
	// TODO: Document this
	Error string `json:"error,omitempty"`
}

// EnvironmentState is a "bill of materials" describing what was deployed to an
// Environment.
type EnvironmentState struct {
	// ID is a unique, system-assigned identifier for this state.
	ID string `json:"id,omitempty"`
	// GitCommit describes a specific Git repository commit that was used in this
	// state.
	GitCommit *GitCommit `json:"gitCommit,omitempty"`
	// Images describes container images and versions thereof that were used
	// in this state.
	Images []Image `json:"images,omitempty"`
	// Charts describes Helm charts that were used in this state.
	Charts []Chart `json:"charts,omitempty"`
	// HealthCheckCommit is the ID of a specific commit in the Environment's Git
	// repository. When determining environment health checks, associated Argo CD
	// Application resources will be checked not only for their own health status,
	// but also to see whether they are synced to this specific commit.
	HealthCheckCommit string `json:"healthCheckCommit,omitempty"`
	// Health is the state's last observed health. If this state is the
	// Environment's current state, this will be continuously re-assessed and
	// updated. If this state is a past state of the Environment, this field will
	// denote the last observed health state before transitioning into a different
	// state.
	Health *Health `json:"health,omitempty"`
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
	if !e.GitCommit.Equals(rhs.GitCommit) {
		return false
	}
	if len(e.Images) != len(rhs.Images) {
		return false
	}
	if len(e.Charts) != len(rhs.Charts) {
		return false
	}
	// The order of images shouldn't matter, we have some work to do to make an
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
	// The order of charts shouldn't matter, we have some work to do to make an
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
