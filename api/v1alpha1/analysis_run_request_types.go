package v1alpha1

import (
	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:resource:scope=Namespaced,shortName={analysisrunrequest,analysisrunrequests}
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name=Stage,type=string,JSONPath=`.spec.stage`
// +kubebuilder:printcolumn:name=Freight,type=string,JSONPath=`.spec.freight`
// +kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:subresource:status
type AnalysisRunRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec AnalysisRunRequestSpec `json:"spec"`

	// +kubebuilder:validation:Optional
	Status AnalysisRunRequestStatus `json:"status,omitempty"`
}

type AnalysisRunRequestSpec struct {
	// StageName is the name of initiating stage
	// +kubebuilder:validation:Required
	StageName string `json:"stageName"`
	// PromotionName (optional) is the name of the last promotion being verified
	// +optional
	PromotionName string `json:"promotionName,omitempty"`
	// RequestedFreight spec copied from Stage spec
	// +kubebuilder:validation:Required
	RequestedFreight []FreightRequest `json:"requestedFreight"`
	// StageVars copied from Stage spec
	// +optional
	StageVars []ExpressionVariable `json:"stageVars,omitempty"`
	// FreightCollection to verify. Stripped from verification history
	// +kubebuilder:validation:Required
	FreightCollection FreightCollection `json:"freightCollection"`
	// Copy of the Stage.Spec.Verification.
	// +kubebuilder:validation:Required
	VerificationTemplate Verification `json:"verificationTemplate"`
	// Targets describes a list of targets to run AnalysisRuns for
	// +listType=atomic
	// +kubebuilder:validation:Required
	Targets []PromotionRequestTarget `json:"targets"`

	// UpdateStrategy configures the pace of running AnalysisRuns
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf"
	UpdateStrategy TargetUpdateStrategy `json:"updateStrategy,omitempty"`
}

type AnalysisRunRequestPhase string

const (
	AnalysisRunRequestPhasePending      AnalysisRunRequestPhase = "Pending"
	AnalysisRunRequestPhaseRunning      AnalysisRunRequestPhase = "Running"
	AnalysisRunRequestPhaseSuccessful   AnalysisRunRequestPhase = "Successful"
	AnalysisRunRequestPhaseFailed       AnalysisRunRequestPhase = "Failed"
	AnalysisRunRequestPhaseError        AnalysisRunRequestPhase = "Error"
	AnalysisRunRequestPhaseInconclusive AnalysisRunRequestPhase = "Inconclusive"
)

func (phase AnalysisRunRequestPhase) IsTerminal() bool {
	switch phase {
	case AnalysisRunRequestPhaseSuccessful,
		AnalysisRunRequestPhaseFailed,
		AnalysisRunRequestPhaseError,
		AnalysisRunRequestPhaseInconclusive:
		return true
	default:
		return false
	}
}

type AnalysisRunRequestStatus struct {
	Phase AnalysisRunRequestPhase `json:"phase"`
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`
	// +kubebuilder:validation:Optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// +kubebuilder:validation:Optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// +kubebuilder:validation:Optional
	// +listType=atomic
	Targets []AnalysisRunRequestTargetStatus `json:"analysisRuns,omitempty"`
	// FIXME: do we want to aggregate summaries and metrics results here?
}

type AnalysisRunRequestTargetStatus struct {
	// Name is the name of the Target.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	TargetName string `json:"targetName"`
	// +kubebuilder:validation:Optional
	AnalysisRunName string `json:"analysisRunName"`
	// +kubebuilder:validation:Optional
	Status *rolloutsapi.AnalysisRunStatus `json:"status,omitempty"`
}

// VerificationTemplate is a verification spec copied from Stage spec
type VerificationTemplate struct {
	// AnalysisTemplates is a list of AnalysisTemplates from which AnalysisRuns
	// should be created to verify a Stage's current Freight is fit to be promoted
	// downstream.
	AnalysisTemplates []AnalysisTemplateReference `json:"analysisTemplates,omitempty"`
	// AnalysisRunMetadata contains optional metadata that should be applied to
	// all AnalysisRuns.
	AnalysisRunMetadata *AnalysisRunMetadata `json:"analysisRunMetadata,omitempty"`
	// Args lists arguments that should be added to all AnalysisRuns.
	Args []AnalysisRunArgument `json:"args,omitempty"`
}

// +kubebuilder:object:root=true

// PromotionRequestList contains a list of PromotionRequests.
type AnalysisRunRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AnalysisRunRequest `json:"items"`
}

func (s *AnalysisRunRequest) GetStatus() *AnalysisRunRequestStatus {
	return &s.Status
}
