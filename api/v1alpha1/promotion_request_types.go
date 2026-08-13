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

// PromotionRequest groups the Promotions that fan out a Stage's selected Freight
// to its selected Targets.
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

	// Spec describes the Stage and Freight associated with the PromotionRequest.
	//
	// +kubebuilder:validation:Required
	Spec PromotionRequestSpec `json:"spec"`

	// Status describes the current aggregate state of the PromotionRequest's
	// Promotions.
	//
	// +kubebuilder:validation:Optional
	Status PromotionRequestStatus `json:"status,omitempty"`
}

// GetStatus returns the PromotionRequest's status.
func (p *PromotionRequest) GetStatus() *PromotionRequestStatus {
	return &p.Status
}

// PromotionRequestSpec describes the Stage and Freight associated with a
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

	// Targets specifies the Targets to which this PromotionRequest promotes Freight.
	// Each Target MUST be in the same namespace as the PromotionRequest. The list
	// may be empty, which records that the governing Stage's target selectors
	// matched no Targets at the time the PromotionRequest was created.
	//
	// Target names MUST be unique. This is enforced by the PromotionRequest
	// validating webhook rather than by the schema: the list is immutable, so
	// per-item ownership tracking would only inflate the object's managedFields
	// without ever being used to merge anything into it.
	//
	// +listType=atomic
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf"
	Targets []PromotionRequestTarget `json:"targets"`
}

// PromotionRequestTarget identifies a Target selected by a PromotionRequest.
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
