package v1alpha1

import (
	"crypto/sha1"
	"fmt"
	"slices"
	"strings"

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
	// started.
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
	// VerificationPhaseAborted denotes a verification process that has been
	// aborted.
	VerificationPhaseAborted VerificationPhase = "Aborted"
	// VerificationPhaseInconclusive denotes a verification process that has
	// completed with an inconclusive result.
	VerificationPhaseInconclusive VerificationPhase = "Inconclusive"
)

// IsTerminal returns true if the VerificationPhase is a terminal one.
func (v *VerificationPhase) IsTerminal() bool {
	switch *v {
	case VerificationPhaseSuccessful, VerificationPhaseFailed,
		VerificationPhaseError, VerificationPhaseAborted, VerificationPhaseInconclusive:
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

// +kubebuilder:validation:Enum={Warehouse}
type FreightOriginKind string

const FreightOriginKindWarehouse FreightOriginKind = "Warehouse"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Shard,type=string,JSONPath=`.spec.shard`
// +kubebuilder:printcolumn:name=Current Freight,type=string,JSONPath=`.status.currentFreight.name`
// +kubebuilder:printcolumn:name=Health,type=string,JSONPath=`.status.health.status`
// +kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Stage is the Kargo API's main type.
type Stage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes sources of Freight used by the Stage and how to incorporate
	// Freight into the Stage.
	//
	// +kubebuilder:validation:Required
	Spec StageSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
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
	// shard. A defaulting webhook will sync the value of the
	// kargo.akuity.io/shard label with the value of this field. When this field
	// is empty, the webhook will ensure that label is absent.
	Shard string `json:"shard,omitempty" protobuf:"bytes,4,opt,name=shard"`
	// Subscriptions describes the Stage's sources of Freight. This is a required
	// field.
	//
	// Deprecated: Use RequestedFreight instead.
	Subscriptions Subscriptions `json:"subscriptions" protobuf:"bytes,1,opt,name=subscriptions"`
	// RequestedFreight expresses the Stage's need for certain pieces of Freight,
	// each having originated from a particular Warehouse. This list must be
	// non-empty. In the common case, a Stage will request Freight having
	// originated from just one specific Warehouse. In advanced cases, requesting
	// Freight from multiple Warehouses provides a method of advancing new
	// artifacts of different types through parallel pipelines at different
	// speeds. This can be useful, for instance, if a Stage is home to multiple
	// microservices that are independently versioned.
	//
	// +kubebuilder:validation:MinItems=1
	RequestedFreight []FreightRequest `json:"requestedFreight" protobuf:"bytes,5,rep,name=requestedFreight"`
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
//
// Deprecated: Use FreightRequest instead.
type Subscriptions struct {
	// Warehouse is a subscription to a Warehouse. This field is mutually
	// exclusive with the UpstreamStages field.
	Warehouse string `json:"warehouse,omitempty" protobuf:"bytes,1,opt,name=warehouse"`
	// UpstreamStages identifies other Stages as potential sources of Freight
	// for this Stage. This field is mutually exclusive with the Repos field.
	UpstreamStages []StageSubscription `json:"upstreamStages,omitempty" protobuf:"bytes,2,rep,name=upstreamStages"`
}

// StageSubscription defines a subscription to Freight from another Stage.
//
// Deprecated: Use FreightRequest instead.
type StageSubscription struct {
	// Name specifies the name of a Stage.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
}

// FreightRequest expresses a Stage's need for Freight having originated from a
// particular Warehouse.
type FreightRequest struct {
	// Origin specifies from where the requested Freight must have originated.
	// This is a required field.
	//
	// +kubebuilder:validation:Required
	Origin FreightOrigin `json:"origin" protobuf:"bytes,1,opt,name=origin"`
	// Sources describes where the requested Freight may be obtained from. This is
	// a required field.
	Sources FreightSources `json:"sources" protobuf:"bytes,2,opt,name=sources"`
}

// FreightOrigin describes a kind of Freight in terms of where it may have
// originated.
//
// +protobuf.options.(gogoproto.goproto_stringer)=false
type FreightOrigin struct {
	// Kind is the kind of resource from which Freight may have originated. At
	// present, this can only be "Warehouse".
	//
	// +kubebuilder:validation:Required
	Kind FreightOriginKind `json:"kind" protobuf:"bytes,1,opt,name=kind"`
	// Name is the name of the resource of the kind indicated by the Kind field
	// from which Freight may originated.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
}

func (f *FreightOrigin) String() string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s", f.Kind, f.Name)
}

func (f *FreightOrigin) Equals(other *FreightOrigin) bool {
	if f == nil && other == nil {
		return true
	}
	if f == nil || other == nil {
		return false
	}
	return f.Kind == other.Kind && f.Name == other.Name
}

type FreightSources struct {
	// Direct indicates the requested Freight may be obtained directly from the
	// Warehouse from which it originated. If this field's value is false, then
	// the value of the Stages field must be non-empty. i.e. Between the two
	// fields, at least one source must be specified.
	Direct bool `json:"direct,omitempty" protobuf:"varint,1,opt,name=direct"`
	// Stages identifies other "upstream" Stages as potential sources of the
	// requested Freight. If this field's value is empty, then the value of the
	// Direct field must be true. i.e. Between the two fields, at least on source
	// must be specified.
	Stages []string `json:"stages,omitempty" protobuf:"bytes,2,rep,name=stages"`
}

// PromotionMechanisms describes how to incorporate Freight into a Stage.
type PromotionMechanisms struct {
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. Its value is overridable by
	// child promotion mechanisms.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,3,opt,name=origin"`
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
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^https?://(\w+([\.-]\w+)*@)?\w+([\.-]\w+)*(:[\d]+)?(/.*)?$`
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, the branch
	// checked out by this promotion mechanism will be the one specified by the
	// ReadBranch field. If that, too, is unspecified, the default branch of the
	// repository will be checked out. Always provide a value for this field if
	// wishing to check out a specific commit indicated by a piece of Freight.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,9,opt,name=origin"`
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
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^(\w+([-/]\w+)*)?$`
	ReadBranch string `json:"readBranch,omitempty" protobuf:"bytes,3,opt,name=readBranch"`
	// WriteBranch specifies the particular branch of the repository to be
	// updated. This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^\w+([-/]\w+)*$`
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
	// GitLab indicates git provider is GitLab
	GitLab *GitLabPullRequest `json:"gitlab,omitempty" protobuf:"bytes,2,opt,name=gitlab"`
}

type GitHubPullRequest struct {
}

type GitLabPullRequest struct {
}

// KargoRenderPromotionMechanism describes how to use Kargo Render to
// incorporate Freight into a Stage.
type KargoRenderPromotionMechanism struct {
	// Images describes how images can be incorporated into a Stage using Kargo
	// Render. If this field is omitted, all images in the Freight being promoted
	// will be passed to Kargo Render in the form <image name>:<tag>. (e.g. Will
	// not use digests by default.)
	//
	// +kubebuilder:validation:Optional
	Images []KargoRenderImageUpdate `json:"images,omitempty" protobuf:"bytes,1,rep,name=images"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing GitRepoUpdate's Origin field.
	// If that, too, is unspecified, Promotions will fail if there is ever
	// ambiguity regarding from which piece of Freight an artifact is to be
	// sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,2,opt,name=origin"`
}

// KargoRenderImageUpdate describes how an image can be incorporated into a
// Stage using Kargo Render.
type KargoRenderImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing
	// KargoRenderPromotionMechanism's Origin field. If that, too, is unspecified,
	// Promotions will fail if there is ever ambiguity regarding from which piece
	// of Freight an artifact is to be sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,3,opt,name=origin"`
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	// +kubebuilder:validation:Optional
	UseDigest bool `json:"useDigest" protobuf:"varint,2,opt,name=useDigest"`
}

// KustomizePromotionMechanism describes how to use Kustomize to incorporate
// Freight into a Stage.
type KustomizePromotionMechanism struct {
	// Images describes images for which `kustomize edit set image` should be
	// executed and the paths in which those commands should be executed.
	//
	// +kubebuilder:validation:MinItems=1
	Images []KustomizeImageUpdate `json:"images" protobuf:"bytes,1,rep,name=images"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing GitRepoUpdate's Origin field.
	// If that, too, is unspecified, Promotions will fail if there is ever
	// ambiguity regarding from which piece of Freight an artifact is to be
	// sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,2,opt,name=origin"`
}

// KustomizeImageUpdate describes how to run `kustomize edit set image`
// for a given image.
type KustomizeImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing KustomizePromotionMechanism's
	// Origin field. If that, too, is unspecified, Promotions will fail if there
	// is ever ambiguity regarding from which piece of Freight an artifact is to
	// be sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,4,opt,name=origin"`
	// Path specifies a path in which the `kustomize edit set image` command
	// should be executed. This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	Path string `json:"path" protobuf:"bytes,2,opt,name=path"`
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	// +kubebuilder:validation:Optional
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
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing GitRepoUpdate's Origin field.
	// If that, too, is unspecified, Promotions will fail if there is ever
	// ambiguity regarding from which piece of Freight an artifact is to be
	// sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,3,opt,name=origin"`
}

// HelmImageUpdate describes how a specific image version can be incorporated
// into a specific Helm values file.
type HelmImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^(\w+([\.-]\w+)*(:[\d]+)?/)?(\w+([\.-]\w+)*)(/\w+([\.-]\w+)*)*$`
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing HelmPromotionMechanism's
	// Origin field. If that, too, is unspecified, Promotions will fail if there
	// is ever ambiguity regarding from which piece of Freight an artifact is to
	// be sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,5,opt,name=origin"`
	// ValuesFilePath specifies a path to the Helm values file that is to be
	// updated. This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	ValuesFilePath string `json:"valuesFilePath" protobuf:"bytes,2,opt,name=valuesFilePath"`
	// Key specifies a key within the Helm values file that is to be updated. This
	// is a required field.
	//
	// +kubebuilder:validation:MinLength=1
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
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^(((https?)|(oci))://)([\w\d\.\-]+)(:[\d]+)?(/.*)*$`
	Repository string `json:"repository" protobuf:"bytes,1,opt,name=repository"`
	// Name along with Repository identifies a subchart of the umbrella chart at
	// ChartPath whose version should be updated. The values of both fields should
	// exactly match the values of the fields of the same names in a dependency
	// expressed in the Chart.yaml of the umbrella chart at ChartPath. i.e. Do not
	// match the values of these two fields to your Warehouse; match them to the
	// Chart.yaml. This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing HelmPromotionMechanism's
	// Origin field. If that, too, is unspecified, Promotions will fail if there
	// is ever ambiguity regarding from which piece of Freight an artifact is to
	// be sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,4,opt,name=origin"`
	// ChartPath is the path to an umbrella chart.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[\w-\.]+(/[\w-\.]+)*$
	ChartPath string `json:"chartPath" protobuf:"bytes,3,opt,name=chartPath"`
}

// ArgoCDAppUpdate describes updates that should be applied to an Argo CD
// Application resources to incorporate Freight into a Stage.
type ArgoCDAppUpdate struct {
	// AppName specifies the name of an Argo CD Application resource to be
	// updated.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppName string `json:"appName" protobuf:"bytes,1,opt,name=appName"`
	// AppNamespace specifies the namespace of an Argo CD Application resource to
	// be updated. If left unspecified, the namespace of this Application resource
	// will use the value of ARGOCD_NAMESPACE or "argocd"
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	AppNamespace string `json:"appNamespace,omitempty" protobuf:"bytes,2,opt,name=appNamespace"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional, but Promotions will fail if there
	// is ever ambiguity regarding which piece of Freight from which an artifact
	// is to be sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,4,opt,name=origin"`
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
	// +kubebuilder:validation:MinLength=1
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// Chart along with the RepoURL field identifies which of an Argo CD
	// Application's sources this update is intended for. Note: As of Argo CD 2.6,
	// Applications can use multiple sources. When the source to be updated
	// references a Helm chart repository, the values of the RepoURL and Chart
	// fields should exactly match the values of the fields of the same names in
	// the source. i.e. Do not match the values of these two fields to your
	// Warehouse; match them to the Application source you wish to update.
	//
	// +kubebuilder:validation:Optional
	Chart string `json:"chart,omitempty" protobuf:"bytes,2,opt,name=chart"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing ArgoCDAppUpdate's Origin
	// field. If that, too, is unspecified, Promotions will fail if there is ever
	// ambiguity regarding from which piece of Freight an artifact is to be
	// sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,6,opt,name=origin"`
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
	// +kubebuilder:validation:MinItems=1
	Images []ArgoCDKustomizeImageUpdate `json:"images" protobuf:"bytes,1,rep,name=images"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing ArgoCDSourceUpdate's Origin
	// field. If that, too, is unspecified, Promotions will fail if there is ever
	// ambiguity regarding from which piece of Freight an artifact is to be
	// sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,2,opt,name=origin"`
}

// ArgoCDHelm describes updates to an Argo CD Application source's Helm-specific
// attributes to incorporate newly observed Freight into a Stage.
type ArgoCDHelm struct {
	// Images describes how specific image versions can be incorporated into an
	// Argo CD Application's Helm parameters.
	//
	// +kubebuilder:validation:MinItems=1
	Images []ArgoCDHelmImageUpdate `json:"images" protobuf:"bytes,1,rep,name=images"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing ArgoCDSourceUpdate's Origin
	// field. If that, too, is unspecified, Promotions will fail if there is ever
	// ambiguity regarding from which piece of Freight an artifact is to be
	// sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,2,opt,name=origin"`
}

// ArgoCDKustomizeImageUpdate describes how a specific image version can be
// incorporated into an Argo CD Application's Kustomize parameters.
type ArgoCDKustomizeImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing ArgoCDKustomize's Origin
	// field. If that, too, is unspecified, Promotions will fail if there is ever
	// ambiguity regarding from which piece of Freight an artifact is to be
	// sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,3,opt,name=origin"`
	// UseDigest specifies whether the image's digest should be used instead of
	// its tag.
	//
	// +kubebuilder:validation:Optional
	UseDigest bool `json:"useDigest" protobuf:"varint,2,opt,name=useDigest"`
}

// ArgoCDHelmImageUpdate describes how a specific image version can be
// incorporated into an Argo CD Application's Helm parameters.
type ArgoCDHelmImageUpdate struct {
	// Image specifies a container image (without tag). This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// Origin disambiguates the origin from which artifacts used by this promotion
	// mechanism must have originated. This is especially useful in cases where a
	// Stage may request Freight from multiples origins (e.g. multiple Warehouses)
	// and some of those each reference different versions of artifacts from the
	// same repository. This field is optional. When left unspecified, it will
	// implicitly inherit the value of the enclosing ArgoCDHelm's Origin field. If
	// that, too, is unspecified, Promotions will fail if there is ever ambiguity
	// regarding from which piece of Freight an artifact is to be sourced.
	Origin *FreightOrigin `json:"origin,omitempty" protobuf:"bytes,4,opt,name=origin"`
	// Key specifies a key within an Argo CD Application's Helm parameters that is
	// to be updated. This is a required field.
	//
	// +kubebuilder:validation:MinLength=1
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
	// LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh
	// annotation that was handled by the controller. This field can be used to
	// determine whether the request to refresh the resource has been handled.
	// +optional
	LastHandledRefresh string `json:"lastHandledRefresh,omitempty" protobuf:"bytes,11,opt,name=lastHandledRefresh"`
	// Phase describes where the Stage currently is in its lifecycle.
	Phase StagePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`
	// FreightHistory is a list of recent Freight selections that were deployed
	// to the Stage. By default, the last ten Freight selections are stored.
	// The first item in the list is the most recent Freight selection and
	// currently deployed to the Stage, subsequent items are older selections.
	FreightHistory FreightHistory `json:"freightHistory,omitempty" protobuf:"bytes,4,rep,name=freightHistory" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// CurrentFreight is a simplified representation of the Stage's current
	// Freight describing what is currently deployed to the Stage.
	//
	// Deprecated: Use the top item in the FreightHistory stack instead.
	CurrentFreight *FreightReference `json:"currentFreight,omitempty" protobuf:"bytes,2,opt,name=currentFreight"`
	// History is a stack of recent Freight. By default, the last ten Freight are
	// stored.
	//
	// Deprecated: Use the FreightHistory stack instead.
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
	CurrentPromotion *PromotionReference `json:"currentPromotion,omitempty" protobuf:"bytes,7,opt,name=currentPromotion"`
	// LastPromotion is a reference to the last completed promotion.
	LastPromotion *PromotionReference `json:"lastPromotion,omitempty" protobuf:"bytes,10,opt,name=lastPromotion"`
}

// FreightReference is a simplified representation of a piece of Freight -- not
// a root resource type.
type FreightReference struct {
	// Name is system-assigned identifier that is derived deterministically from
	// the contents of the Freight. i.e. Two pieces of Freight can be compared for
	// equality by comparing their Names.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Warehouse is the name of the Warehouse that created this Freight.
	//
	// Deprecated: Use the Origin instead.
	Warehouse string `json:"warehouse,omitempty" protobuf:"bytes,6,opt,name=warehouse"`
	// Origin describes a kind of Freight in terms of its origin.
	Origin FreightOrigin `json:"origin,omitempty" protobuf:"bytes,8,opt,name=origin"`
	// Commits describes specific Git repository commits.
	Commits []GitCommit `json:"commits,omitempty" protobuf:"bytes,2,rep,name=commits"`
	// Images describes specific versions of specific container images.
	Images []Image `json:"images,omitempty" protobuf:"bytes,3,rep,name=images"`
	// Charts describes specific versions of specific Helm charts.
	Charts []Chart `json:"charts,omitempty" protobuf:"bytes,4,rep,name=charts"`
	// VerificationInfo is information about any verification process that was
	// associated with this Freight for this Stage.
	//
	// Deprecated: Use FreightCollection.VerificationHistory instead.
	VerificationInfo *VerificationInfo `json:"verificationInfo,omitempty" protobuf:"bytes,5,opt,name=verificationInfo"`
	// VerificationHistory is a stack of recent VerificationInfo. By default,
	// the last ten VerificationInfo are stored.
	//
	// Deprecated: Use FreightCollection.VerificationHistory instead.
	VerificationHistory VerificationInfoStack `json:"verificationHistory,omitempty" protobuf:"bytes,7,rep,name=verificationHistory"`
}

// FreightCollection is a collection of FreightReferences, each of which
// represents a piece of Freight that has been selected for deployment to a
// Stage.
type FreightCollection struct {
	// ID is a unique and deterministically calculated identifier for the
	// FreightCollection. It is updated on each use of the UpdateOrPush method.
	ID string `json:"id" protobuf:"bytes,3,opt,name=id"`
	// Freight is a map of FreightReference objects, indexed by their Warehouse
	// origin.
	Freight map[string]FreightReference `json:"items,omitempty" protobuf:"bytes,1,rep,name=items" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// VerificationHistory is a stack of recent VerificationInfo. By default,
	// the last ten VerificationInfo are stored.
	VerificationHistory VerificationInfoStack `json:"verificationHistory,omitempty" protobuf:"bytes,2,rep,name=verificationHistory"`
}

// UpdateOrPush updates the entry in the FreightCollection based on the
// Warehouse name of the provided FreightReference. If no such entry exists, the
// provided FreightReference is appended to the FreightCollection. This function
// is not concurrency-safe.
func (f *FreightCollection) UpdateOrPush(freight ...FreightReference) {
	if f.Freight == nil {
		f.Freight = make(map[string]FreightReference, len(freight))
	}
	for _, i := range freight {
		f.Freight[i.Origin.String()] = i
	}
	freightNames := make([]string, 0, len(f.Freight))
	for _, freight := range f.Freight {
		freightNames = append(freightNames, freight.Name)
	}
	slices.Sort(freightNames)
	f.ID = fmt.Sprintf("%x", sha1.Sum([]byte(strings.Join(freightNames, ","))))
}

// References returns a slice of FreightReference objects from the
// FreightCollection. The slice is ordered by the origin of the
// FreightReference objects.
func (f *FreightCollection) References() []FreightReference {
	if f == nil || len(f.Freight) == 0 {
		return nil
	}

	var origins []string
	for o := range f.Freight {
		origins = append(origins, o)
	}
	slices.Sort(origins)

	var refs []FreightReference
	for _, o := range origins {
		refs = append(refs, f.Freight[o])
	}
	return refs
}

// FreightHistory is a linear list of FreightCollection items. The list is
// ordered by the time at which the FreightCollection was recorded, with the
// most recent (current) FreightCollection at the top of the list.
type FreightHistory []*FreightCollection

// Current returns the most recent (current) FreightCollection from the history.
func (f *FreightHistory) Current() *FreightCollection {
	if f == nil || len(*f) == 0 {
		return nil
	}
	return (*f)[0]
}

// Record appends the provided FreightCollection as the most recent (current)
// FreightCollection in the history. I.e. The provided FreightCollection becomes
// the first item in the list. If the list grows beyond ten items, the bottom
// items are removed.
func (f *FreightHistory) Record(freight ...*FreightCollection) {
	*f = append(freight, *f...)
	f.truncate()
}

// truncate ensures the history does not grow beyond 10 items.
func (f *FreightHistory) truncate() {
	const maxSize = 10
	if f != nil && len(*f) > maxSize {
		*f = (*f)[:maxSize]
	}
}

// FreightReferenceStack is a linear stack of FreightReferences.
//
// Deprecated: Use FreightHistory instead.
type FreightReferenceStack []FreightReference

// Push appends the provided FreightReference to the top of the stack. If the
// stack grows beyond 10 items, the bottom items are removed.
func (f *FreightReferenceStack) Push(freight ...FreightReference) {
	*f = append(freight, *f...)
	f.truncate()
}

// UpdateOrPush updates the first FreightReference with the same name as the
// provided FreightReference or appends the provided FreightReference to the
// stack if no such FreightReference is found.
//
// The order of existing items in the stack is preserved, and new items without
// a matching name are appended to the top of the stack. If the stack grows
// beyond 10 items, the bottom items are removed.
func (f *FreightReferenceStack) UpdateOrPush(freight ...FreightReference) {
	var newStack FreightReferenceStack
	for _, i := range freight {
		var found bool
		for fi, item := range *f {
			if i.Name == item.Name {
				(*f)[fi] = i
				found = true
				break
			}
		}
		if !found {
			newStack = append(newStack, i)
		}
	}

	*f = append(newStack, *f...)
	f.truncate()
}

func (f *FreightReferenceStack) truncate() {
	const maxSize = 10
	if len(*f) > maxSize {
		*f = (*f)[:maxSize]
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

// +kubebuilder:object:root=true

// StageList is a list of Stage resources.
type StageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Stage `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type PromotionReference struct {
	// Name is the name of the Promotion
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Freight is the freight being promoted
	Freight FreightReference `json:"freight" protobuf:"bytes,2,opt,name=freight"`
	// Status is the (optional) status of the promotion
	Status *PromotionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
	// FinishedAt is the time at which the Promotion was completed.
	FinishedAt *metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,4,opt,name=finishedAt"`
}

// Verification describes how to verify that a Promotion has been successful
// using Argo Rollouts AnalysisTemplates.
type Verification struct {
	// AnalysisTemplates is a list of AnalysisTemplates from which AnalysisRuns
	// should be created to verify a Stage's current Freight is fit to be promoted
	// downstream.
	AnalysisTemplates []AnalysisTemplateReference `json:"analysisTemplates,omitempty" protobuf:"bytes,1,rep,name=analysisTemplates"`
	// AnalysisRunMetadata contains optional metadata that should be applied to
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
	// +kubebuilder:validation:Required
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
	// +kubebuilder:validation:Required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Value is the value of the argument.
	//
	// +kubebuilder:validation:Required
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
}

// VerificationInfo contains the details of an instance of a Verification
// process.
type VerificationInfo struct {
	// ID is the identifier of the Verification process.
	ID string `json:"id,omitempty" protobuf:"bytes,4,opt,name=id"`
	// Actor is the name of the entity that initiated or aborted the
	// Verification process.
	Actor string `json:"actor,omitempty" protobuf:"bytes,7,opt,name=actor"`
	// StartTime is the time at which the Verification process was started.
	StartTime *metav1.Time `json:"startTime,omitempty" protobuf:"bytes,5,opt,name=startTime"`
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
	// FinishTime is the time at which the Verification process finished.
	FinishTime *metav1.Time `json:"finishTime,omitempty" protobuf:"bytes,6,opt,name=finishTime"`
}

// HasAnalysisRun returns a bool indicating whether the VerificationInfo has an
// associated AnalysisRun.
func (v *VerificationInfo) HasAnalysisRun() bool {
	return v != nil && v.AnalysisRun != nil
}

type VerificationInfoStack []VerificationInfo

// Current returns the VerificationInfo at the top of the stack.
func (v *VerificationInfoStack) Current() *VerificationInfo {
	if len(*v) == 0 {
		return nil
	}
	return &(*v)[0]
}

// UpdateOrPush updates the VerificationInfo with the same ID as the provided
// VerificationInfo or appends the provided VerificationInfo to the stack if no
// such VerificationInfo is found.
//
// The order of existing items in the stack is preserved, and new items without
// a matching ID are appended to the top of the stack. If the stack grows beyond
// 10 items, the bottom items are removed.
func (v *VerificationInfoStack) UpdateOrPush(info ...VerificationInfo) {
	var newStack VerificationInfoStack
	for _, i := range info {
		var found bool
		for vi, item := range *v {
			if i.ID == item.ID {
				(*v)[vi] = i
				found = true
				break
			}
		}
		if !found {
			newStack = append(newStack, i)
		}
	}

	*v = append(newStack, *v...)

	const maxSize = 10
	if len(*v) > maxSize {
		*v = (*v)[:maxSize]
	}
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
