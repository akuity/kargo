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

// PromotionRequest expresses the intent to promote a piece of Freight to the
// Targets its Stage governs. It is a living object: the governing Stage owns
// its target selectors and keeps them in sync with the Stage's own, and its
// status reflects the current per-Target promotion state as Targets come and
// go -- closer to a ReplicaSet than to a Job.
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

	// Spec describes the Stage, the Freight, and the target selectors of the
	// PromotionRequest.
	//
	// +kubebuilder:validation:Required
	Spec PromotionRequestSpec `json:"spec"`

	// Status describes the current resolution of the target selectors and the
	// aggregate state of the PromotionRequest's Promotions.
	//
	// +kubebuilder:validation:Optional
	Status PromotionRequestStatus `json:"status,omitempty"`
}

// GetStatus returns the PromotionRequest's status.
func (p *PromotionRequest) GetStatus() *PromotionRequestStatus {
	return &p.Status
}

// PromotionRequestSpec describes the Stage, the Freight, and the target
// selectors of a PromotionRequest.
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

	// TargetSelectors select the Targets to which this PromotionRequest promotes
	// Freight, matching Targets by their labels within the PromotionRequest's
	// own Project. A Target is selected when it matches any selector in this
	// list. An empty selector in a non-empty list selects all Targets in the
	// Project; a list that matches nothing leaves the PromotionRequest with
	// nothing to do.
	//
	// The governing Stage owns this field: the selectors are copied from the
	// Stage's own at creation and kept in sync when they change. Unlike Stage
	// and Freight, this field is expected to change over the PromotionRequest's
	// lifetime; the reconciler responds by creating Promotions for newly
	// selected Targets and recording resolution in status.
	//
	// +optional
	TargetSelectors []metav1.LabelSelector `json:"targetSelectors,omitempty"`
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

	// Targets records the current resolution of the target selectors: one entry
	// per selected Target, with the child Promotion promoting to it and that
	// Promotion's phase. Entries come and go as the selectors and the Project's
	// Targets change.
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
