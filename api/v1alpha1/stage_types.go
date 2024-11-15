package v1alpha1

import (
	"crypto/sha1"
	"fmt"
	"slices"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	// StagePhaseFailed denotes a Stage that is in a failed state. For example,
	// the Stage may have failed to promote or verify its Freight.
	StagePhaseFailed StagePhase = "Failed"
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
	HealthStateHealthy       HealthState = "Healthy"
	HealthStateNotApplicable HealthState = "NotApplicable"
	HealthStateProgressing   HealthState = "Progressing"
	HealthStateUnknown       HealthState = "Unknown"
	HealthStateUnhealthy     HealthState = "Unhealthy"
)

var stateOrder = map[HealthState]int{
	HealthStateHealthy:       0,
	HealthStateNotApplicable: 1,
	HealthStateProgressing:   2,
	HealthStateUnknown:       3,
	HealthStateUnhealthy:     4,
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
// +kubebuilder:printcolumn:name=Current Freight,type=string,JSONPath=`.status.freightSummary`
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

// IsControlFlow returns true if the Stage is a control flow Stage. A control
// flow Stage is one that does not incorporate Freight into itself, but rather
// orchestrates the promotion of Freight from one or more upstream Stages to
// one or more downstream Stages.
func (s *Stage) IsControlFlow() bool {
	switch {
	case s.Spec.PromotionTemplate != nil && len(s.Spec.PromotionTemplate.Spec.Steps) > 0:
		return false
	default:
		return true
	}
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
	// PromotionTemplate describes how to incorporate Freight into the Stage
	// using a Promotion.
	PromotionTemplate *PromotionTemplate `json:"promotionTemplate,omitempty" protobuf:"bytes,6,opt,name=promotionTemplate"`
	// Verification describes how to verify a Stage's current Freight is fit for
	// promotion downstream.
	Verification *Verification `json:"verification,omitempty" protobuf:"bytes,3,opt,name=verification"`
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

// PromotionTemplate defines a template for a Promotion that can be used to
// incorporate Freight into a Stage.
type PromotionTemplate struct {
	Spec PromotionTemplateSpec `json:"spec" protobuf:"bytes,1,opt,name=spec"`
}

// PromotionTemplateSpec describes the (partial) specification of a Promotion
// for a Stage. This is a template that can be used to create a Promotion for a
// Stage.
type PromotionTemplateSpec struct {
	// Vars is a list of variables that can be referenced by expressions in
	// promotion steps.
	Vars []PromotionVariable `json:"vars,omitempty" protobuf:"bytes,2,rep,name=vars"`
	// Steps specifies the directives to be executed as part of a Promotion.
	// The order in which the directives are executed is the order in which they
	// are listed in this field.
	//
	// +kubebuilder:validation:MinItems=1
	Steps []PromotionStep `json:"steps,omitempty" protobuf:"bytes,1,rep,name=steps"`
}

// StageStatus describes a Stages's current and recent Freight, health, and
// more.
type StageStatus struct {
	// Conditions contains the last observations of the Stage's current
	// state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge" protobuf:"bytes,13,rep,name=conditions"`
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
	// FreightSummary is human-readable text maintained by the controller that
	// summarizes what Freight is currently deployed to the Stage. For Stages that
	// request a single piece of Freight AND the request has been fulfilled, this
	// field will simply contain the name of the Freight. For Stages that request
	// a single piece of Freight AND the request has NOT been fulfilled, or for
	// Stages that request multiple pieces of Freight, this field will contain a
	// summary of fulfilled/requested Freight. The existence of this field is a
	// workaround for kubectl limitations so that this complex but valuable
	// information can be displayed in a column in response to `kubectl get
	// stages`.
	FreightSummary string `json:"freightSummary,omitempty" protobuf:"bytes,12,opt,name=freightSummary"`
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

func (w *StageStatus) GetConditions() []metav1.Condition {
	return w.Conditions
}

func (w *StageStatus) SetConditions(conditions []metav1.Condition) {
	w.Conditions = conditions
}

// FreightReference is a simplified representation of a piece of Freight -- not
// a root resource type.
type FreightReference struct {
	// Name is system-assigned identifier that is derived deterministically from
	// the contents of the Freight. i.e. Two pieces of Freight can be compared for
	// equality by comparing their Names.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Origin describes a kind of Freight in terms of its origin.
	Origin FreightOrigin `json:"origin,omitempty" protobuf:"bytes,8,opt,name=origin"`
	// Commits describes specific Git repository commits.
	Commits []GitCommit `json:"commits,omitempty" protobuf:"bytes,2,rep,name=commits"`
	// Images describes specific versions of specific container images.
	Images []Image `json:"images,omitempty" protobuf:"bytes,3,rep,name=images"`
	// Charts describes specific versions of specific Helm charts.
	Charts []Chart `json:"charts,omitempty" protobuf:"bytes,4,rep,name=charts"`
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

// HasNonTerminalVerification returns true if the FreightCollection has any
// verification which is not in a terminal state, indicating verification is
// still in progress.
func (f *FreightCollection) HasNonTerminalVerification() bool {
	for _, v := range f.VerificationHistory {
		if !v.Phase.IsTerminal() {
			return true
		}
	}
	return false
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

// DeepEquals returns a bool indicating whether the receiver deep-equals the
// provided Image. I.e., all fields must be equal.
func (i *Image) DeepEquals(other *Image) bool {
	if i == nil && other == nil {
		return true
	}
	if i == nil || other == nil {
		return false
	}
	return i.RepoURL == other.RepoURL &&
		i.GitRepoURL == other.GitRepoURL &&
		i.Tag == other.Tag &&
		i.Digest == other.Digest
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

// DeepEquals returns a bool indicating whether the receiver deep-equals the
// provided Chart. I.e., all fields must be equal.
func (c *Chart) DeepEquals(other *Chart) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.RepoURL == other.RepoURL &&
		c.Name == other.Name &&
		c.Version == other.Version
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
	// Config is the opaque configuration of all health checks performed on this
	// Stage.
	Config *apiextensionsv1.JSON `json:"config,omitempty" protobuf:"bytes,4,opt,name=config"`
	// Output is the opaque output of all health checks performed on this Stage.
	Output *apiextensionsv1.JSON `json:"output,omitempty" protobuf:"bytes,5,opt,name=output"`
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

// PromotionReference contains the relevant information about a Promotion
// as observed by a Stage.
type PromotionReference struct {
	// Name is the name of the Promotion.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Freight is the freight being promoted.
	Freight *FreightReference `json:"freight,omitempty" protobuf:"bytes,2,opt,name=freight"`
	// Status is the (optional) status of the Promotion.
	Status *PromotionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
	// FinishedAt is the time at which the Promotion was completed.
	FinishedAt *metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,4,opt,name=finishedAt"`
}

// GetHealthChecks returns the list of health checks for the PromotionReference.
func (r *PromotionReference) GetHealthChecks() []HealthCheckStep {
	if r == nil || r.Status == nil {
		return nil
	}
	return r.Status.HealthChecks
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
