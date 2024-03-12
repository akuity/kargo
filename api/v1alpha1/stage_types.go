package v1alpha1

import (
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
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes sources of Freight used by the Stage and how to incorporate
	// Freight into the Stage.
	//
	//+kubebuilder:validation:Required
	Spec *StageSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the Stage's current and recent Freight, health, and more.
	Status StageStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

func (s *Stage) GetStatus() *StageStatus {
	return &s.Status
}

// StageSpec describes the sources of Freight used by a Stage and how to
// incorporate Freight into the Stage.
type StageSpec struct {
	// Shard is the name of the shard that this Stage belongs to. This is an
	// optional field. If not specified, the Stage will belong to the default
	// shard. A defaulting webhook will sync this field with the value of the
	// kargo.akuity.io/shard label. When the shard label is not present or differs
	// from the value of this field, the defaulting webhook will set the label to
	// the value of this field. If the shard label is present and this field is
	// empty, the defaulting webhook will set the value of this field to the value
	// of the shard label.
	Shard string `json:"shard,omitempty" protobuf:"bytes,4,opt,name=shard"`
	// Subscriptions describes the Stage's sources of Freight. This is a required
	// field.
	//
	//+kubebuilder:validation:Required
	Subscriptions *Subscriptions `json:"subscriptions" protobuf:"bytes,1,opt,name=subscriptions"`
	// PromotionMechanisms describes how to incorporate Freight into the Stage.
	// This is an optional field as it is sometimes useful to aggregates available
	// Freight from multiple upstream Stages without performing any actions. The
	// utility of this is to allow multiple downstream Stages to subscribe to a
	// single upstream Stage where they may otherwise have subscribed to multiple
	// upstream Stages.
	PromotionMechanisms *PromotionMechanisms `json:"promotionMechanisms,omitempty" protobuf:"bytes,2,opt,name=promotionMechanisms"`
	// Verification describes how to verify a Stage's current Freight is fit for
	// promotion downstream.
	Verification *Verification `json:"verification,omitempty" protobuf:"bytes,3,opt,name=verification"`
}

// Subscriptions describes a Stage's sources of Freight.
type Subscriptions struct {
	// Warehouse is a subscription to a Warehouse. This field is mutually
	// exclusive with the UpstreamStages field.
	Warehouse string `json:"warehouse,omitempty" protobuf:"bytes,1,opt,name=warehouse"`
	// UpstreamStages identifies other Stages as potential sources of Freight
	// for this Stage. This field is mutually exclusive with the Repos field.
	UpstreamStages []StageSubscription `json:"upstreamStages,omitempty" protobuf:"bytes,2,rep,name=upstreamStages"`
}

// StageSubscription defines a subscription to Freight from another Stage.
type StageSubscription struct {
	// Name specifies the name of a Stage.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
}

// PromotionMechanisms describes how to incorporate Freight into a Stage.
type PromotionMechanisms struct {
	// GitRepoUpdates describes updates that should be applied to Git repositories
	// to incorporate Freight into the Stage. This field is optional, as such
	// actions are not required in all cases.
	GitRepoUpdates []GitRepoUpdate `json:"gitRepoUpdates,omitempty" protobuf:"bytes,1,rep,name=gitRepoUpdates"`
	// ArgoCDAppUpdates describes updates that should be applied to Argo CD
	// Application resources to incorporate Freight into the Stage. This field is
	// optional, as such actions are not required in all cases. Note that all
	// updates specified by the GitRepoUpdates field, if any, are applied BEFORE
	// these.
	ArgoCDAppUpdates []ArgoCDAppUpdate `json:"argoCDAppUpdates,omitempty" protobuf:"bytes,2,rep,name=argoCDAppUpdates"`
}

// GitRepoUpdate describes updates that should be applied to a Git repository
// (using various configuration management tools) to incorporate Freight into a
// Stage.
type GitRepoUpdate struct {
	// RepoURL is the URL of the repository to update. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^https?://(\w+([\.-]\w+)*@)?\w+([\.-]\w+)*(:[\d]+)?(/.*)?$`
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when connecting to the repository. This should be enabled
	// only with great caution.
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty" protobuf:"varint,2,opt,name=insecureSkipTLSVerify"`
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
	ReadBranch string `json:"readBranch,omitempty" protobuf:"bytes,3,opt,name=readBranch"`
	// WriteBranch specifies the particular branch of the repository to be
	// updated. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^\w+([-/]\w+)*$`
	WriteBranch string `json:"writeBranch" protobuf:"bytes,4,opt,name=writeBranch"`
	// PullRequest will generate a pull request instead of making the commit directly
	PullRequest *PullRequestPromotionMechanism `json:"pullRequest,omitempty" protobuf:"bytes,5,opt,name=pullRequest"`
	// Render describes how to use Kargo Render to incorporate Freight into the
	// Stage. This is mutually exclusive with the Kustomize and Helm fields.
	Render *KargoRenderPromotionMechanism `json:"render,omitempty" protobuf:"bytes,6,opt,name=render"`
	// Kustomize describes how to use Kustomize to incorporate Freight into the
	// Stage. This is mutually exclusive with the Render and Helm fields.
	Kustomize *KustomizePromotionMechanism `json:"kustomize,omitempty" protobuf:"bytes,7,opt,name=kustomize"`
	// Helm describes how to use Helm to incorporate Freight into the Stage. This
	// is mutually exclusive with the Render and Kustomize fields.
	Helm *HelmPromotionMechanism `json:"helm,omitempty" protobuf:"bytes,8,opt,name=helm"`
}

// PullRequestPromotionMechanism describes how to generate a pull request against the write branch during promotion
// Attempts to infer the git provider from well-known git domains.
type PullRequestPromotionMechanism struct {
	// GitHub indicates git provider is GitHub
	GitHub *GitHubPullRequest `json:"github,omitempty" protobuf:"bytes,1,opt,name=github"`
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
	Images []KargoRenderImageUpdate `json:"images,omitempty" protobuf:"bytes,1,rep,name=images"`
}

// KargoRenderImageUpdate describes how an image can be incorporated into a
// Stage using Kargo Render.
type KargoRenderImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	//+kubebuilder:validation:Optional
	UseDigest bool `json:"useDigest" protobuf:"varint,2,opt,name=useDigest"`
}

// KustomizePromotionMechanism describes how to use Kustomize to incorporate
// Freight into a Stage.
type KustomizePromotionMechanism struct {
	// Images describes images for which `kustomize edit set image` should be
	// executed and the paths in which those commands should be executed.
	//
	//+kubebuilder:validation:MinItems=1
	Images []KustomizeImageUpdate `json:"images" protobuf:"bytes,1,rep,name=images"`
}

// KustomizeImageUpdate describes how to run `kustomize edit set image`
// for a given image.
type KustomizeImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// Path specifies a path in which the `kustomize edit set image` command
	// should be executed. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	Path string `json:"path" protobuf:"bytes,2,opt,name=path"`
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	//+kubebuilder:validation:Optional
	UseDigest bool `json:"useDigest" protobuf:"varint,3,opt,name=useDigest"`
}

// HelmPromotionMechanism describes how to use Helm to incorporate Freight into
// a Stage.
type HelmPromotionMechanism struct {
	// Images describes how specific image versions can be incorporated into Helm
	// values files.
	Images []HelmImageUpdate `json:"images,omitempty" protobuf:"bytes,1,rep,name=images"`
	// Charts describes how specific chart versions can be incorporated into an
	// umbrella chart.
	Charts []HelmChartDependencyUpdate `json:"charts,omitempty" protobuf:"bytes,2,rep,name=charts"`
}

// HelmImageUpdate describes how a specific image version can be incorporated
// into a specific Helm values file.
type HelmImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(\w+([\.-]\w+)*(:[\d]+)?/)?(\w+([\.-]\w+)*)(/\w+([\.-]\w+)*)*$`
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// ValuesFilePath specifies a path to the Helm values file that is to be
	// updated. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	ValuesFilePath string `json:"valuesFilePath" protobuf:"bytes,2,opt,name=valuesFilePath"`
	// Key specifies a key within the Helm values file that is to be updated. This
	// is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Key string `json:"key" protobuf:"bytes,3,opt,name=key"`
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
	Value ImageUpdateValueType `json:"value" protobuf:"bytes,4,opt,name=value"`
}

// HelmChartDependencyUpdate describes how a specific Helm chart that is used
// as a subchart of an umbrella chart can be updated.
type HelmChartDependencyUpdate struct {
	// Repository along with Name identifies a subchart of the umbrella chart at
	// ChartPath whose version should be updated. The values of both fields should
	// exactly match the values of the fields of the same names in a dependency
	// expressed in the Chart.yaml of the umbrella chart at ChartPath. i.e. Do not
	// match the values of these two fields to your Warehouse; match them to the
	// Chart.yaml. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=`^(((https?)|(oci))://)([\w\d\.\-]+)(:[\d]+)?(/.*)*$`
	Repository string `json:"repository" protobuf:"bytes,1,opt,name=repository"`
	// Name along with Repository identifies a subchart of the umbrella chart at
	// ChartPath whose version should be updated. The values of both fields should
	// exactly match the values of the fields of the same names in a dependency
	// expressed in the Chart.yaml of the umbrella chart at ChartPath. i.e. Do not
	// match the values of these two fields to your Warehouse; match them to the
	// Chart.yaml. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// ChartPath is the path to an umbrella chart.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	ChartPath string `json:"chartPath" protobuf:"bytes,3,opt,name=chartPath"`
}

// ArgoCDAppUpdate describes updates that should be applied to an Argo CD
// Application resources to incorporate Freight into a Stage.
type ArgoCDAppUpdate struct {
	// AppName specifies the name of an Argo CD Application resource to be
	// updated.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppName string `json:"appName" protobuf:"bytes,1,opt,name=appName"`
	// AppNamespace specifies the namespace of an Argo CD Application resource to
	// be updated. If left unspecified, the namespace of this Application resource
	// will use the value of ARGOCD_NAMESPACE or "argocd"
	//
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppNamespace string `json:"appNamespace,omitempty" protobuf:"bytes,2,opt,name=appNamespace"`
	// SourceUpdates describes updates to be applied to various sources of the
	// specified Argo CD Application resource.
	SourceUpdates []ArgoCDSourceUpdate `json:"sourceUpdates,omitempty" protobuf:"bytes,3,rep,name=sourceUpdates"`
}

// ArgoCDSourceUpdate describes updates that should be applied to one of an Argo
// CD Application resource's sources.
type ArgoCDSourceUpdate struct {
	// RepoURL along with the Chart field identifies which of an Argo CD
	// Application's sources this update is intended for. Note: As of Argo CD 2.6,
	// Applications can use multiple sources. When the source to be updated
	// references a Helm chart repository, the values of the RepoURL and Chart
	// fields should exactly match the values of the fields of the same names in
	// the source. i.e. Do not match the values of these two fields to your
	// Warehouse; match them to the Application source you wish to update. This is
	// a required field.
	//
	//+kubebuilder:validation:MinLength=1
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// Chart along with the RepoURL field identifies which of an Argo CD
	// Application's sources this update is intended for. Note: As of Argo CD 2.6,
	// Applications can use multiple sources. When the source to be updated
	// references a Helm chart repository, the values of the RepoURL and Chart
	// fields should exactly match the values of the fields of the same names in
	// the source. i.e. Do not match the values of these two fields to your
	// Warehouse; match them to the Application source you wish to update.
	//
	//+kubebuilder:validation:Optional
	Chart string `json:"chart,omitempty" protobuf:"bytes,2,opt,name=chart"`
	// UpdateTargetRevision is a bool indicating whether the source should be
	// updated such that its TargetRevision field points at the most recently git
	// commit (if RepoURL references a git repository) or chart version (if
	// RepoURL references a chart repository).
	UpdateTargetRevision bool `json:"updateTargetRevision,omitempty" protobuf:"varint,3,opt,name=updateTargetRevision"`
	// Kustomize describes updates to the source's Kustomize-specific attributes.
	Kustomize *ArgoCDKustomize `json:"kustomize,omitempty" protobuf:"bytes,4,opt,name=kustomize"`
	// Helm describes updates to the source's Helm-specific attributes.
	Helm *ArgoCDHelm `json:"helm,omitempty" protobuf:"bytes,5,opt,name=helm"`
}

// ArgoCDKustomize describes updates to an Argo CD Application source's
// Kustomize-specific attributes to incorporate newly observed Freight into a
// Stage.
type ArgoCDKustomize struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Kustomize parameters.
	//
	//+kubebuilder:validation:MinItems=1
	Images []ArgoCDKustomizeImageUpdate `json:"images" protobuf:"bytes,1,rep,name=images"`
}

// ArgoCDHelm describes updates to an Argo CD Application source's Helm-specific
// attributes to incorporate newly observed Freight into a Stage.
type ArgoCDHelm struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Helm parameters.
	//
	//+kubebuilder:validation:MinItems=1
	Images []ArgoCDHelmImageUpdate `json:"images" protobuf:"bytes,1,rep,name=images"`
}

// ArgoCDKustomizeImageUpdate describes how a specific image version can be
// incorporated into an Argo CD Application's Kustomize parameters.
type ArgoCDKustomizeImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	//+kubebuilder:validation:Optional
	UseDigest bool `json:"useDigest" protobuf:"varint,2,opt,name=useDigest"`
}

// ArgoCDHelmImageUpdate describes how a specific image version can be
// incorporated into an Argo CD Application's Helm parameters.
type ArgoCDHelmImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// Key specifies a key within an Argo CD Application's Helm parameters that is
	// to be updated. This is a required field.
	//
	//+kubebuilder:validation:MinLength=1
	Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
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
	Value ImageUpdateValueType `json:"value" protobuf:"bytes,3,opt,name=value"`
}

// StageStatus describes a Stages's current and recent Freight, health, and
// more.
type StageStatus struct {
	// Phase describes where the Stage currently is in its lifecycle.
	Phase StagePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`
	// CurrentFreight is a simplified representation of the Stage's current
	// Freight describing what is currently deployed to the Stage.
	CurrentFreight *FreightReference `json:"currentFreight,omitempty" protobuf:"bytes,2,opt,name=currentFreight"`
	// History is a stack of recent Freight. By default, the last ten Freight are
	// stored.
	History FreightReferenceStack `json:"history,omitempty" protobuf:"bytes,3,rep,name=history"`
	// Health is the Stage's last observed health.
	Health *Health `json:"health,omitempty" protobuf:"bytes,8,opt,name=health"`
	// Message describes any errors that are preventing the Stage controller
	// from assessing Stage health or from finding new Freight.
	Message string `json:"message,omitempty" protobuf:"bytes,9,opt,name=message"`
	// ObservedGeneration represents the .metadata.generation that this Stage
	// status was reconciled against.
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,6,opt,name=observedGeneration"`
	// CurrentPromotion is a reference to the currently Running promotion.
	CurrentPromotion *PromotionInfo `json:"currentPromotion,omitempty" protobuf:"bytes,7,opt,name=currentPromotion"`
}

// FreightReference is a simplified representation of a piece of Freight -- not
// a root resource type.
type FreightReference struct {
	// ID is system-assigned value that is derived deterministically from the
	// contents of the Freight. i.e. Two pieces of Freight can be compared for
	// equality by comparing their IDs.
	ID string `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	// Commits describes specific Git repository commits.
	Commits []GitCommit `json:"commits,omitempty" protobuf:"bytes,2,rep,name=commits"`
	// Images describes specific versions of specific container images.
	Images []Image `json:"images,omitempty" protobuf:"bytes,3,rep,name=images"`
	// Charts describes specific versions of specific Helm charts.
	Charts []Chart `json:"charts,omitempty" protobuf:"bytes,4,rep,name=charts"`
	// VerificationInfo is information about any verification process that was
	// associated with this Freight for this Stage.
	VerificationInfo *VerificationInfo `json:"verificationInfo,omitempty" protobuf:"bytes,5,opt,name=verificationInfo"`
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
	RepoURL string `json:"repoURL,omitempty" protobuf:"bytes,1,opt,name=repoURL"`
	// GitRepoURL specifies the URL of a Git repository that contains the source
	// code for the image repository referenced by the RepoURL field if Kargo was
	// able to infer it.
	GitRepoURL string `json:"gitRepoURL,omitempty" protobuf:"bytes,2,opt,name=gitRepoURL"`
	// Tag identifies a specific version of the image in the repository specified
	// by RepoURL.
	Tag string `json:"tag,omitempty" protobuf:"bytes,3,opt,name=tag"`
	// Digest identifies a specific version of the image in the repository
	// specified by RepoURL. This is a more precise identifier than Tag.
	Digest string `json:"digest,omitempty" protobuf:"bytes,4,opt,name=digest"`
}

// Chart describes a specific version of a Helm chart.
type Chart struct {
	// RepoURL specifies the URL of a Helm chart repository. Classic chart
	// repositories (using HTTP/S) can contain differently named charts. When this
	// field points to such a repository, the Name field will specify the name of
	// the chart within the repository. In the case of a repository within an OCI
	// registry, the URL implicitly points to a specific chart and the Name field
	// will be empty.
	RepoURL string `json:"repoURL,omitempty" protobuf:"bytes,1,opt,name=repoURL"`
	// Name specifies the name of the chart.
	Name string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	// Version specifies a particular version of the chart.
	Version string `json:"version,omitempty" protobuf:"bytes,3,opt,name=version"`
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
	Status HealthState `json:"status,omitempty" protobuf:"bytes,1,opt,name=status"`
	// Issues clarifies why a Stage in any state other than Healthy is in that
	// state. This field will always be the empty when a Stage is Healthy.
	Issues []string `json:"issues,omitempty" protobuf:"bytes,2,rep,name=issues"`
	// ArgoCDApps describes the current state of any related ArgoCD Applications.
	ArgoCDApps []ArgoCDAppStatus `json:"argoCDApps,omitempty" protobuf:"bytes,3,rep,name=argoCDApps"`
}

// ArgoCDAppStatus describes the current state of a single ArgoCD Application.
type ArgoCDAppStatus struct {
	// Namespace is the namespace of the ArgoCD Application.
	Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	// Name is the name of the ArgoCD Application.
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// HealthStatus is the health of the ArgoCD Application.
	HealthStatus ArgoCDAppHealthStatus `json:"healthStatus,omitempty" protobuf:"bytes,3,opt,name=healthStatus"`
	// SyncStatus is the sync status of the ArgoCD Application.
	SyncStatus ArgoCDAppSyncStatus `json:"syncStatus,omitempty" protobuf:"bytes,4,opt,name=syncStatus"`
}

// ArgoCDAppHealthStatus describes the health of an ArgoCD Application.
type ArgoCDAppHealthStatus struct {
	Status  ArgoCDAppHealthState `json:"status" protobuf:"bytes,1,opt,name=status"`
	Message string               `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
}

// ArgoCDAppSyncStatus describes the sync status of an ArgoCD Application.
type ArgoCDAppSyncStatus struct {
	Status    ArgoCDAppSyncState `json:"status" protobuf:"bytes,1,opt,name=status"`
	Revision  string             `json:"revision,omitempty" protobuf:"bytes,2,opt,name=revision"`
	Revisions []string           `json:"revisions,omitempty" protobuf:"bytes,3,rep,name=revisions"`
}

//+kubebuilder:object:root=true

// StageList is a list of Stage resources.
type StageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Stage `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type PromotionInfo struct {
	// Name is the name of the Promotion
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Freight is the freight being promoted
	Freight FreightReference `json:"freight" protobuf:"bytes,2,opt,name=freight"`
}

// Verification describes how to verify that a Promotion has been successful
// using Argo Rollouts AnalysisTemplates.
type Verification struct {
	// AnalysisTemplates is a list of AnalysisTemplates from which AnalysisRuns
	// should be created to verify a Stage's current Freight is fit to be promoted
	// downstream.
	AnalysisTemplates []AnalysisTemplateReference `json:"analysisTemplates,omitempty" protobuf:"bytes,1,rep,name=analysisTemplates"`
	// AnalysisRunMetadata is contains optional metadata that should be applied to
	// all AnalysisRuns.
	AnalysisRunMetadata *AnalysisRunMetadata `json:"analysisRunMetadata,omitempty" protobuf:"bytes,2,opt,name=analysisRunMetadata"`
	// Args lists arguments that should be added to all AnalysisRuns.
	Args []AnalysisRunArgument `json:"args,omitempty" protobuf:"bytes,3,rep,name=args"`
}

// AnalysisTemplateReference is a reference to an AnalysisTemplate.
type AnalysisTemplateReference struct {
	// Name is the name of the AnalysisTemplate in the same project/namespace as
	// the Stage.
	//
	//+kubebuilder:validation:Required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
}

// AnalysisRunMetadata contains optional metadata that should be applied to all
// AnalysisRuns.
type AnalysisRunMetadata struct {
	// Additional labels to apply to an AnalysisRun.
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,1,rep,name=labels" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Additional annotations to apply to an AnalysisRun.
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,2,rep,name=annotations" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

// AnalysisRunArgument represents an argument to be added to an AnalysisRun.
type AnalysisRunArgument struct {
	// Name is the name of the argument.
	//
	//+kubebuilder:validation:Required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Value is the value of the argument.
	//
	//+kubebuilder:validation:Required
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
}

// VerificationInfo contains information about the currently running
// Verification process.
type VerificationInfo struct {
	// Phase describes the current phase of the Verification process. Generally,
	// this will be a reflection of the underlying AnalysisRun's phase, however,
	// there are exceptions to this, such as in the case where an AnalysisRun
	// cannot be launched successfully.
	Phase VerificationPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`
	// Message may contain additional information about why the verification
	// process is in its current phase.
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	// AnalysisRun is a reference to the Argo Rollouts AnalysisRun that implements
	// the Verification process.
	AnalysisRun *AnalysisRunReference `json:"analysisRun,omitempty" protobuf:"bytes,3,opt,name=analysisRun"`
}

// AnalysisRunReference is a reference to an AnalysisRun.
type AnalysisRunReference struct {
	// Namespace is the namespace of the AnalysisRun.
	Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	// Name is the name of the AnalysisRun.
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// Phase is the last observed phase of the AnalysisRun referenced by Name.
	Phase string `json:"phase" protobuf:"bytes,3,opt,name=phase"`
}
