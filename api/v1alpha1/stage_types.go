package v1alpha1

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StagePhase string

const (
	// StagePhaseNotApplicable denotes a Stage that has no Freight.
	StagePhaseNotApplicable StagePhase = "NotApplicable"
	// StagePhaseSteady denotes a Stage that has Freight and is not currently
	// being promoted or verified.
	StagePhaseSteady StagePhase = "Steady"
	// StagePhasePromoting denotes a Stage that is currently being promoted.
	StagePhasePromoting StagePhase = "Promoting"
	// StagePhaseVerifying denotes a Stage that is currently being verified.
	StagePhaseVerifying StagePhase = "Verifying"
)

type VerificationPhase string

// Note: VerificationPhases are identical to AnalysisRunPhases. In almost all
// cases, the VerificationPhase will be a reflection of the underlying
// AnalysisRunPhase. There are exceptions to this, such as in the case where an
// AnalysisRun cannot be launched successfully.

const (
	// VerificationPhasePending denotes a verification process that has not yet
	// started yet.
	VerificationPhasePending VerificationPhase = "Pending"
	// VerificationPhaseRunning denotes a verification that is currently running.
	VerificationPhaseRunning VerificationPhase = "Running"
	// VerificationPhaseSuccessful denotes a verification process that has
	// completed successfully.
	VerificationPhaseSuccessful VerificationPhase = "Successful"
	// VerificationPhaseFailed denotes a verification process that has completed
	// with a failure.
	VerificationPhaseFailed VerificationPhase = "Failed"
	// VerificationPhaseError denotes a verification process that has completed
	// with an error.
	VerificationPhaseError VerificationPhase = "Error"
	// VerificationPhaseInconclusive denotes a verification process that has
	// completed with an inconclusive result.
	VerificationPhaseInconclusive VerificationPhase = "Inconclusive"
)

// IsTerminal returns true if the VerificationPhase is a terminal one.
func (v *VerificationPhase) IsTerminal() bool {
	switch *v {
	case VerificationPhaseSuccessful, VerificationPhaseFailed,
		VerificationPhaseError, VerificationPhaseInconclusive:
		return true
	default:
		return false
	}
}

// +kubebuilder:validation:Enum={ImageAndTag,Tag,ImageAndDigest,Digest}
type ImageUpdateValueType string

const (
	ImageUpdateValueTypeImageAndTag    ImageUpdateValueType = "ImageAndTag"
	ImageUpdateValueTypeTag            ImageUpdateValueType = "Tag"
	ImageUpdateValueTypeImageAndDigest ImageUpdateValueType = "ImageAndDigest"
	ImageUpdateValueTypeDigest         ImageUpdateValueType = "Digest"
)

type HealthState string

const (
	HealthStateHealthy     HealthState = "Healthy"
	HealthStateUnhealthy   HealthState = "Unhealthy"
	HealthStateProgressing HealthState = "Progressing"
	HealthStateUnknown     HealthState = "Unknown"
)

var stateOrder = map[HealthState]int{
	HealthStateHealthy:     0,
	HealthStateProgressing: 1,
	HealthStateUnknown:     2,
	HealthStateUnhealthy:   3,
}

// Merge returns the more severe of two HealthStates.
func (h HealthState) Merge(other HealthState) HealthState {
	if stateOrder[h] > stateOrder[other] {
		return h
	}
	return other
}

type ArgoCDAppHealthState string

const (
	ArgoCDAppHealthStateUnknown     ArgoCDAppHealthState = "Unknown"
	ArgoCDAppHealthStateProgressing ArgoCDAppHealthState = "Progressing"
	ArgoCDAppHealthStateHealthy     ArgoCDAppHealthState = "Healthy"
	ArgoCDAppHealthStateSuspended   ArgoCDAppHealthState = "Suspended"
	ArgoCDAppHealthStateDegraded    ArgoCDAppHealthState = "Degraded"
	ArgoCDAppHealthStateMissing     ArgoCDAppHealthState = "Missing"
)

type ArgoCDAppSyncState string

const (
	ArgoCDAppSyncStateUnknown   ArgoCDAppSyncState = "Unknown"
	ArgoCDAppSyncStateSynced    ArgoCDAppSyncState = "Synced"
	ArgoCDAppSyncStateOutOfSync ArgoCDAppSyncState = "OutOfSync"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name=Current Freight,type=string,JSONPath=`.status.currentFreight.id`
//+kubebuilder:printcolumn:name=Health,type=string,JSONPath=`.status.health.status`
//+kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Stage is the Kargo API's main type.
type Stage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes sources of Freight used by the Stage and how to incorporate
	// Freight into the Stage.
	//
	//+kubebuilder:validation:Required
	Spec *StageSpec `json:"spec"`
	// Status describes the Stage's current and recent Freight, health, and more.
	Status StageStatus `json:"status,omitempty"`
}

func (s *Stage) GetStatus() *StageStatus {
	return &s.Status
}

// StageSpec describes the sources of Freight used by a Stage and how to
// incorporate Freight into the Stage.
type StageSpec struct {
	// Subscriptions describes the Stage's sources of Freight. This is a required
	// field.
	//
	//+kubebuilder:validation:Required
	Subscriptions *Subscriptions `json:"subscriptions"`
	// PromotionMechanisms describes how to incorporate Freight into the Stage.
	// This is an optional field as it is sometimes useful to aggregates available
	// Freight from multiple upstream Stages without performing any actions. The
	// utility of this is to allow multiple downstream Stages to subscribe to a
	// single upstream Stage where they may otherwise have subscribed to multiple
	// upstream Stages.
	PromotionMechanisms *PromotionMechanisms `json:"promotionMechanisms,omitempty"`
	// Verification describes how to verify a Stage's current Freight is fit for
	// promotion downstream.
	Verification *Verification `json:"verification,omitempty"`
}

// Subscriptions describes a Stage's sources of Freight.
type Subscriptions struct {
	// Warehouse is a subscription to a Warehouse. This field is mutually
	// exclusive with the UpstreamStages field.
	Warehouse string `json:"warehouse,omitempty"`
	// UpstreamStages identifies other Stages as potential sources of Freight
	// for this Stage. This field is mutually exclusive with the Repos field.
	UpstreamStages []StageSubscription `json:"upstreamStages,omitempty"`
}

// StageSubscription defines a subscription to Freight from another Stage.
type StageSubscription struct {
	// Name specifies the name of a Stage.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name"`
}

// PromotionMechanisms describes how to incorporate Freight into a Stage.
type PromotionMechanisms struct {
	// GitRepoUpdates describes updates that should be applied to Git repositories
	// to incorporate Freight into the Stage. This field is optional, as such
	// actions are not required in all cases.
	GitRepoUpdates []GitRepoUpdate `json:"gitRepoUpdates,omitempty"`
	// ArgoCDAppUpdates describes updates that should be applied to Argo CD
	// Application resources to incorporate Freight into the Stage. This field is
	// optional, as such actions are not required in all cases. Note that all
	// updates specified by the GitRepoUpdates field, if any, are applied BEFORE
	// these.
	ArgoCDAppUpdates []ArgoCDAppUpdate `json:"argoCDAppUpdates,omitempty"`
}

// GitRepoUpdate describes updates that should be applied to a Git repository
// (using various configuration management tools) to incorporate Freight into a
// Stage.
type GitRepoUpdate struct {
	// RepoURL is the URL of the repository to update. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^https://(\w+([\.-]\w+)*@)?\w+([\.-]\w+)*(:[\d]+)?(/.*)?$`
	RepoURL string `json:"repoURL"`
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when connecting to the repository. This should be enabled
	// only with great caution.
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
	// ReadBranch specifies a particular branch of the repository from which to
	// locate contents that will be written to the branch specified by the
	// WriteBranch field. This field is optional. When not specified, the
	// ReadBranch is implicitly the repository's default branch AND in cases where
	// a Freight includes a GitCommit, that commit's ID will supersede the value
	// of this field. Therefore, in practice, this field is only used to clarify
	// what branch of a repository can be treated as a source of manifests or
	// other configuration when a Stage has no subscription to that repository.
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=`^(\w+([-/]\w+)*)?$`
	ReadBranch string `json:"readBranch,omitempty"`
	// WriteRepoURL specifies the particular repo to write to on the specified WriteBranch
	// updated. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^https://(\w+([\.-]\w+)*@)?\w+([\.-]\w+)*(:[\d]+)?(/.*)?$`
	WriteRepoURL string `json:"writeRepoURL"`
	// WriteBranch specifies the particular branch of the repository to be
	// updated. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^\w+([-/]\w+)*$`
	WriteBranch string `json:"writeBranch"`
	// PullRequest will generate a pull request instead of making the commit directly
	PullRequest *PullRequestPromotionMechanism `json:"pullRequest,omitempty"`
	// Render describes how to use Kargo Render to incorporate Freight into the
	// Stage. This is mutually exclusive with the Kustomize and Helm fields.
	Render *KargoRenderPromotionMechanism `json:"render,omitempty"`
	// Kustomize describes how to use Kustomize to incorporate Freight into the
	// Stage. This is mutually exclusive with the Render and Helm fields.
	Kustomize *KustomizePromotionMechanism `json:"kustomize,omitempty"`
	// Helm describes how to use Helm to incorporate Freight into the Stage. This
	// is mutually exclusive with the Render and Kustomize fields.
	Helm *HelmPromotionMechanism `json:"helm,omitempty"`
}

// PullRequestPromotionMechanism describes how to generate a pull request against the write branch during promotion
// Attempts to infer the git provider from well-known git domains.
type PullRequestPromotionMechanism struct {
	// GitHub indicates git provider is GitHub
	GitHub *GitHubPullRequest `json:"github,omitempty"`
}

type GitHubPullRequest struct {
}

// KargoRenderPromotionMechanism describes how to use Kargo Render to
// incorporate Freight into a Stage.
type KargoRenderPromotionMechanism struct {
	// Images describes how images can be incorporated into a Stage using Kargo
	// Render. If this field is omitted, all images in the Freight being promoted
	// will be passed to Kargo Render in the form <image name>:<tag>. (e.g. Will
	// not use digests by default.)
	//
	//+kubebuilder:validation:Optional
	Images []KargoRenderImageUpdate `json:"images"`
}

// KargoRenderImageUpdate describes how an image can be incorporated into a
// Stage using Kargo Render.
type KargoRenderImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	//+kubebuilder:validation:Optional
	UseDigest bool `json:"useDigest"`
}

// KustomizePromotionMechanism describes how to use Kustomize to incorporate
// Freight into a Stage.
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
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	//+kubebuilder:validation:Optional
	UseDigest bool `json:"useDigest"`
}

// HelmPromotionMechanism describes how to use Helm to incorporate Freight into
// a Stage.
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
	//+kubebuilder:validation:Pattern=`^(\w+([\.-]\w+)*(:[\d]+)?/)?(\w+([\.-]\w+)*)(/\w+([\.-]\w+)*)*$`
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
	// values file. Valid values are:
	//
	// - ImageAndTag: Replaces the value of the specified key with
	//   <image name>:<tag>
	// - Tag: Replaces the value of the specified key with just the new tag
	// - ImageAndDigest: Replaces the value of the specified key with
	//   <image name>@<digest>
	// - Digest: Replaces the value of the specified key with just the new digest.
	//
	// This is a required field.
	Value ImageUpdateValueType `json:"value"`
}

// HelmChartDependencyUpdate describes how a specific Helm chart that is used
// as a subchart of an umbrella chart can be updated.
type HelmChartDependencyUpdate struct {
	// RegistryURL along with Name identify a subchart of the umbrella chart at
	// ChartPath whose version should be updated.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(((https?)|(oci))://)([\w\d\.\-]+)(:[\d]+)?(/.*)*$`
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
// Application resources to incorporate Freight into a Stage.
type ArgoCDAppUpdate struct {
	// AppName specifies the name of an Argo CD Application resource to be
	// updated.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppName string `json:"appName"`
	// AppNamespace specifies the namespace of an Argo CD Application resource to
	// be updated. If left unspecified, the namespace of this Application resource
	// will use the value of ARGOCD_NAMESPACE or "argocd"
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppNamespace string `json:"appNamespace,omitempty"`
	// SourceUpdates describes updates to be applied to various sources of the
	// specified Argo CD Application resource.
	SourceUpdates []ArgoCDSourceUpdate `json:"sourceUpdates,omitempty"`
}

func (a *ArgoCDAppUpdate) AppNamespaceOrDefault() string {
	if a.AppNamespace != "" {
		return a.AppNamespace
	}
	if envArgocdNs := os.Getenv("ARGOCD_NAMESPACE"); envArgocdNs != "" {
		return envArgocdNs
	}
	return "argocd"
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
// Kustomize-specific attributes to incorporate newly observed Freight into a
// Stage.
type ArgoCDKustomize struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Kustomize parameters.
	//
	//+kubebuilder:validation:MinItems=1
	Images []ArgoCDKustomizeImageUpdate `json:"images"`
}

// ArgoCDHelm describes updates to an Argo CD Application source's Helm-specific
// attributes to incorporate newly observed Freight into a Stage.
type ArgoCDHelm struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Helm parameters.
	//
	//+kubebuilder:validation:MinItems=1
	Images []ArgoCDHelmImageUpdate `json:"images"`
}

// ArgoCDKustomizeImageUpdate describes how a specific image version can be
// incorporated into an Argo CD Application's Kustomize parameters.
type ArgoCDKustomizeImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	//+kubebuilder:validation:Optional
	UseDigest bool `json:"useDigest"`
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
	// Application's Helm parameters. Valid values are:
	//
	// - ImageAndTag: Replaces the value of the specified key with
	//   <image name>:<tag>
	// - Tag: Replaces the value of the specified key with just the new tag
	// - ImageAndDigest: Replaces the value of the specified key with
	//   <image name>@<digest>
	// - Digest: Replaces the value of the specified key with just the new digest.
	//
	// This is a required field.
	Value ImageUpdateValueType `json:"value"`
}

// StageStatus describes a Stages's current and recent Freight, health, and
// more.
type StageStatus struct {
	// Phase describes where the Stage currently is in its lifecycle.
	Phase StagePhase `json:"phase,omitempty"`
	// CurrentFreight is a simplified representation of the Stage's current
	// Freight describing what is currently deployed to the Stage.
	CurrentFreight *FreightReference `json:"currentFreight,omitempty"`
	// History is a stack of recent Freight. By default, the last ten Freight are
	// stored.
	History FreightReferenceStack `json:"history,omitempty"`
	// Health is the Stage's last observed health.
	Health *Health `json:"health,omitempty"`
	// Error describes any errors that are preventing the Stage controller
	// from assessing Stage health or from finding new Freight.
	Error string `json:"error,omitempty"`
	// ObservedGeneration represents the .metadata.generation that this Stage
	// status was reconciled against.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// CurrentPromotion is a reference to the currently Running promotion.
	CurrentPromotion *PromotionInfo `json:"currentPromotion,omitempty"`
}

// FreightReference is a simplified representation of a piece of Freight -- not
// a root resource type.
type FreightReference struct {
	// ID is system-assigned value that is derived deterministically from the
	// contents of the Freight. i.e. Two pieces of Freight can be compared for
	// equality by comparing their IDs.
	ID string `json:"id,omitempty"`
	// Commits describes specific Git repository commits.
	Commits []GitCommit `json:"commits,omitempty"`
	// Images describes specific versions of specific container images.
	Images []Image `json:"images,omitempty"`
	// Charts describes specific versions of specific Helm charts.
	Charts []Chart `json:"charts,omitempty"`
	// VerificationInfo is information about any verification process that was
	// associated with this Freight for this Stage.
	VerificationInfo *VerificationInfo `json:"verificationInfo,omitempty"`
}

type FreightReferenceStack []FreightReference

// Empty returns a bool indicating whether or not the FreightReferenceStack is
// empty. nil counts as empty.
func (f FreightReferenceStack) Empty() bool {
	return len(f) == 0
}

// Pop removes and returns the leading element from a FreightReferenceStack. If
// the FreightReferenceStack is empty, the FreightReferenceStack is not modified
// and a empty FreightReference is returned instead. A boolean is also returned
// indicating whether the returned FreightReference came from the top of the
// stack (true) or is a zero value for that type (false).
func (f *FreightReferenceStack) Pop() (FreightReference, bool) {
	item, ok := f.Top()
	if ok {
		*f = (*f)[1:]
	}
	return item, ok
}

// Top returns the leading element from a FreightReferenceStack without
// modifying the FreightReferenceStack. If the FreightReferenceStack is empty,
// an empty FreightReference is returned instead. A boolean is also returned
// indicating whether the returned FreightReference came from the top of the
// stack (true) or is a zero value for that type (false).
func (f FreightReferenceStack) Top() (FreightReference, bool) {
	if f.Empty() {
		return FreightReference{}, false
	}
	item := *f[0].DeepCopy()
	return item, true
}

// Push pushes one or more Freight onto the FreightStack. The order of
// the new elements at the top of the stack will be equal to the order in which
// they were passed to this function. i.e. The first new element passed will be
// the element at the top of the stack. If resulting modification grow the depth
// of the stack beyond 10 elements, the stack is truncated at the bottom. i.e.
// Modified to contain only the top 10 elements.
func (f *FreightReferenceStack) Push(freight ...FreightReference) {
	*f = append(freight, *f...)
	const max = 10
	if len(*f) > max {
		*f = (*f)[:max]
	}
}

// Image describes a specific version of a container image.
type Image struct {
	// RepoURL describes the repository in which the image can be found.
	RepoURL string `json:"repoURL,omitempty"`
	// GitRepoURL specifies the URL of a Git repository that contains the source
	// code for the image repository referenced by the RepoURL field if Kargo was
	// able to infer it.
	GitRepoURL string `json:"gitRepoURL,omitempty"`
	// Tag identifies a specific version of the image in the repository specified
	// by RepoURL.
	Tag string `json:"tag,omitempty"`
	// Digest identifies a specific version of the image in the repository
	// specified by RepoURL. This is a more precise identifier than Tag.
	Digest string `json:"digest,omitempty"`
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

// Health describes the health of a Stage.
type Health struct {
	// Status describes the health of the Stage.
	Status HealthState `json:"status,omitempty"`
	// Issues clarifies why a Stage in any state other than Healthy is in that
	// state. This field will always be the empty when a Stage is Healthy.
	Issues []string `json:"issues,omitempty"`
	// ArgoCDApps describes the current state of any related ArgoCD Applications.
	ArgoCDApps []ArgoCDAppStatus `json:"argoCDApps,omitempty"`
}

// ArgoCDAppStatus describes the current state of a single ArgoCD Application.
type ArgoCDAppStatus struct {
	// Namespace is the namespace of the ArgoCD Application.
	Namespace string `json:"namespace"`
	// Name is the name of the ArgoCD Application.
	Name string `json:"name"`
	// HealthStatus is the health of the ArgoCD Application.
	HealthStatus ArgoCDAppHealthStatus `json:"healthStatus,omitempty"`
	// SyncStatus is the sync status of the ArgoCD Application.
	SyncStatus ArgoCDAppSyncStatus `json:"syncStatus,omitempty"`
}

// ArgoCDAppHealthStatus describes the health of an ArgoCD Application.
type ArgoCDAppHealthStatus struct {
	Status  ArgoCDAppHealthState `json:"status"`
	Message string               `json:"message,omitempty"`
}

// ArgoCDAppSyncStatus describes the sync status of an ArgoCD Application.
type ArgoCDAppSyncStatus struct {
	Status    ArgoCDAppSyncState `json:"status"`
	Revision  string             `json:"revision,omitempty"`
	Revisions []string           `json:"revisions,omitempty"`
}

//+kubebuilder:object:root=true

// StageList is a list of Stage resources.
type StageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stage `json:"items"`
}

type PromotionInfo struct {
	// Name is the name of the Promotion
	Name string `json:"name"`
	// Freight is the freight being promoted
	Freight FreightReference `json:"freight"`
}

// Verification describes how to verify that a Promotion has been successful
// using Argo Rollouts AnalysisTemplates.
type Verification struct {
	// AnalysisTemplates is a list of AnalysisTemplates from which AnalysisRuns
	// should be created to verify a Stage's current Freight is fit to be promoted
	// downstream.
	AnalysisTemplates []AnalysisTemplateReference `json:"analysisTemplates,omitempty"`
	// AnalysisRunMetadata is contains optional metadata that should be applied to
	// all AnalysisRuns.
	AnalysisRunMetadata *AnalysisRunMetadata `json:"analysisRunMetadata,omitempty"`
	// Args lists arguments that should be added to all AnalysisRuns.
	Args []AnalysisRunArgument `json:"args,omitempty"`
}

// AnalysisTemplateReference is a reference to an AnalysisTemplate.
type AnalysisTemplateReference struct {
	// Name is the name of the AnalysisTemplate in the same project/namespace as
	// the Stage.
	//
	//+kubebuilder:validation:Required
	Name string `json:"name"`
}

// AnalysisRunMetadata contains optional metadata that should be applied to all
// AnalysisRuns.
type AnalysisRunMetadata struct {
	// Additional labels to apply to an AnalysisRun.
	Labels map[string]string `json:"labels,omitempty"`
	// Additional annotations to apply to an AnalysisRun.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// AnalysisRunArgument represents an argument to be added to an AnalysisRun.
type AnalysisRunArgument struct {
	// Name is the name of the argument.
	//
	//+kubebuilder:validation:Required
	Name string `json:"name"`
	// Value is the value of the argument.
	//
	//+kubebuilder:validation:Required
	Value string `json:"value,omitempty"`
}

// VerificationInfo contains information about the currently running
// Verification process.
type VerificationInfo struct {
	// Phase describes the current phase of the Verification process. Generally,
	// this will be a reflection of the underlying AnalysisRun's phase, however,
	// there are exceptions to this, such as in the case where an AnalysisRun
	// cannot be launched successfully.
	Phase VerificationPhase `json:"phase,omitempty"`
	// Message may contain additional information about why the verification
	// process is in its current phase.
	Message string `json:"message,omitempty"`
	// AnalysisRun is a reference to the Argo Rollouts AnalysisRun that implements
	// the Verification process.
	AnalysisRun *AnalysisRunReference `json:"analysisRun,omitempty"`
}

// AnalysisRunReference is a reference to an AnalysisRun.
type AnalysisRunReference struct {
	// Namespace is the namespace of the AnalysisRun.
	Namespace string `json:"namespace"`
	// Name is the name of the AnalysisRun.
	Name string `json:"name"`
	// Phase is the last observed phase of the AnalysisRun referenced by Name.
	Phase string `json:"phase"`
}
