package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PromotionRequestPhasePending denotes a PromotionRequest that has not started
	// creating Promotions.
	PromotionRequestPhasePending PromotionRequestPhase = "Pending"
	// PromotionRequestPhaseRunning denotes a PromotionRequest that is creating or
	// monitoring Promotions.
	PromotionRequestPhaseRunning PromotionRequestPhase = "Running"
	// PromotionRequestPhaseSucceeded denotes a PromotionRequest whose Promotions all
	// completed successfully.
	PromotionRequestPhaseSucceeded PromotionRequestPhase = "Succeeded"
	// PromotionRequestPhaseFailed denotes a PromotionRequest with Promotions that failed
	// for non-technical reasons.
	PromotionRequestPhaseFailed PromotionRequestPhase = "Failed"
	// PromotionRequestPhaseErrored denotes a PromotionRequest with Promotions that
	// encountered technical errors.
	PromotionRequestPhaseErrored PromotionRequestPhase = "Errored"
)

// PromotionRequestPhase is a high-level summary of a PromotionRequest's lifecycle.
type PromotionRequestPhase string

// IsTerminal returns true if the PromotionRequestPhase is a terminal one.
func (p *PromotionRequestPhase) IsTerminal() bool {
	switch *p {
	case PromotionRequestPhaseSucceeded,
		PromotionRequestPhaseFailed,
		PromotionRequestPhaseErrored:
		return true
	default:
		return false
	}
}

// PromotionRequest expresses the intent to promote a piece of Freight to a
// specific set of Targets. Its Stage and Freight are fixed for its lifetime,
// giving it a single freight transition to represent from start to terminal
// state; only its list of Targets may change, so that Targets discovered after
// creation can still be included.
//
// +kubebuilder:resource:scope=Namespaced,shortName={promotionrequest,promotionrequests}
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name=Stage,type=string,JSONPath=`.spec.stage`
// +kubebuilder:printcolumn:name=Freight,type=string,JSONPath=`.spec.freight`
// +kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:subresource:status
type PromotionRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the Stage, the Freight, and the Targets of the
	// PromotionRequest.
	//
	// +kubebuilder:validation:Required
	Spec PromotionRequestSpec `json:"spec"`

	// Status describes the per-Target progress and the aggregate state of the
	// PromotionRequest's Promotions.
	//
	// +kubebuilder:validation:Optional
	Status PromotionRequestStatus `json:"status,omitempty"`
}

// GetStatus returns the PromotionRequest's status.
func (p *PromotionRequest) GetStatus() *PromotionRequestStatus {
	return &p.Status
}

// PromotionRequestSpec describes the Stage, the Freight, and the Targets of a
// PromotionRequest.
type PromotionRequestSpec struct {
	// Stage specifies the name of the Stage that promotes the Freight.
	// The Stage MUST be in the same namespace as the PromotionRequest.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	// +kubebuilder:validation:XValidation:rule="self == oldSelf"
	Stage string `json:"stage"`

	// Freight specifies the piece of Freight promoted by the Stage.
	// The Freight MUST be in the same namespace as the PromotionRequest.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	// +kubebuilder:validation:XValidation:rule="self == oldSelf"
	Freight string `json:"freight"`

	// Targets names the Targets to which this PromotionRequest promotes Freight.
	// Each Target MUST be in the same namespace as the PromotionRequest. The
	// list may be empty, which records that the governing Stage governed no
	// Targets when the PromotionRequest was created -- distinct from the field
	// being absent, which asks for it to be resolved.
	//
	// This is a resolved list, not a selector: the Stage's target selectors are
	// evaluated once, at creation, and the result recorded here. The membership
	// of the PromotionRequest is therefore a snapshot of what the Stage governed
	// at that moment, and its threshold and terminal state are computed against
	// it rather than against a selector that could match differently later.
	//
	// This is the only mutable field in the spec. The governing Stage owns it,
	// and may add Targets to an in-flight PromotionRequest so that Targets
	// discovered after creation can still receive the Freight. Target names MUST
	// be unique; this is enforced by the validating webhook rather than by the
	// schema, since a list-map's per-item ownership tracking would roughly
	// double the storage cost of every entry.
	//
	// +listType=atomic
	// +kubebuilder:validation:Required
	Targets []PromotionRequestTarget `json:"targets"`
}

// PromotionRequestTarget names a Target that a PromotionRequest promotes
// Freight to.
type PromotionRequestTarget struct {
	// Name is the name of the Target.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Name string `json:"name"`
}

// PromotionRequestStatus describes the observed aggregate state of a
// PromotionRequest's Promotions.
type PromotionRequestStatus struct {
	// Conditions contains the last observations of the PromotionRequest's current
	// state.
	//
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge"`

	// Phase is a high-level summary of the PromotionRequest's lifecycle.
	//
	// +kubebuilder:validation:Optional
	Phase PromotionRequestPhase `json:"phase,omitempty"`

	// Targets records progress against spec.targets: one entry per Target, with
	// the child Promotion promoting to it and that Promotion's phase. Entries
	// appear as the reconciler acts on each Target in spec.targets.
	//
	// The list is atomic rather than a map keyed by name: the reconciler is its
	// only writer, so per-item ownership tracking in managedFields would only
	// inflate the object -- roughly doubling the storage cost of each entry --
	// without ever being used to merge.
	//
	// +kubebuilder:validation:Optional
	// +listType=atomic
	Targets []PromotionRequestTargetStatus `json:"targets,omitempty"`

	// Summary aggregates the phases of this PromotionRequest's child Promotions.
	//
	// +kubebuilder:validation:Optional
	Summary *PromotionRequestSummary `json:"summary,omitempty"`

	// ObservedGeneration is the generation of the spec last reconciled.
	//
	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// StartedAt is the time at which the PromotionRequest started.
	//
	// +kubebuilder:validation:Optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// FinishedAt is the time at which the PromotionRequest completed.
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
}

// PromotionRequestTargetStatus records the state of a single Target selected
// by a PromotionRequest.
type PromotionRequestTargetStatus struct {
	// Name is the name of the Target.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Name string `json:"name"`

	// Promotion is the name of the child Promotion currently promoting the
	// Freight to this Target. Empty if none has been created yet.
	//
	// +kubebuilder:validation:Optional
	Promotion string `json:"promotion,omitempty"`

	// Phase is the phase of that Promotion.
	//
	// +kubebuilder:validation:Optional
	Phase PromotionPhase `json:"phase,omitempty"`
}

// PromotionRequestSummary aggregates the phases of a PromotionRequest's child
// Promotions.
type PromotionRequestSummary struct {
	// Pending is the number of child Promotions in the Pending phase.
	Pending int32 `json:"pending,omitempty"`
	// Running is the number of child Promotions in the Running phase.
	Running int32 `json:"running,omitempty"`
	// Succeeded is the number of child Promotions in the Succeeded phase.
	Succeeded int32 `json:"succeeded,omitempty"`
	// Failed is the number of child Promotions in the Failed phase.
	Failed int32 `json:"failed,omitempty"`
	// Errored is the number of child Promotions in the Errored phase.
	Errored int32 `json:"errored,omitempty"`
	// Aborted is the number of child Promotions in the Aborted phase.
	Aborted int32 `json:"aborted,omitempty"`
}

// GetConditions returns the PromotionRequest status conditions.
func (s *PromotionRequestStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}

// SetConditions sets the PromotionRequest status conditions.
func (s *PromotionRequestStatus) SetConditions(conditions []metav1.Condition) {
	s.Conditions = conditions
}

// +kubebuilder:object:root=true

// PromotionRequestList contains a list of PromotionRequests.
type PromotionRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromotionRequest `json:"items"`
}
