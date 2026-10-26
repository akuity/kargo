package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PromotionRollout is not a Kubernetes resource. It lives in Kargo's
// database and is served by the Kargo API server, which is why this file
// carries no CRD markers and the type is not registered with the scheme. It
// keeps the shape of a Kubernetes resource, metadata and all, so that
// clients and the UI can treat it like one. It deliberately embeds no
// TypeMeta: controller-gen takes any type with both TypeMeta and ObjectMeta
// for a kind and would emit a CRD for it.

const (
	// PromotionRolloutPhasePending denotes a PromotionRollout that has not
	// started creating Promotions.
	PromotionRolloutPhasePending PromotionRolloutPhase = "Pending"
	// PromotionRolloutPhaseRunning denotes a PromotionRollout that is creating
	// or monitoring Promotions.
	PromotionRolloutPhaseRunning PromotionRolloutPhase = "Running"
	// PromotionRolloutPhaseSucceeded denotes a PromotionRollout whose
	// Promotions all completed successfully.
	PromotionRolloutPhaseSucceeded PromotionRolloutPhase = "Succeeded"
	// PromotionRolloutPhaseFailed denotes a PromotionRollout with Promotions
	// that failed for non-technical reasons.
	PromotionRolloutPhaseFailed PromotionRolloutPhase = "Failed"
	// PromotionRolloutPhaseErrored denotes a PromotionRollout with Promotions
	// that encountered technical errors.
	PromotionRolloutPhaseErrored PromotionRolloutPhase = "Errored"
	// PromotionRolloutPhaseAborted denotes a PromotionRollout that was
	// aborted before all of its Promotions completed.
	PromotionRolloutPhaseAborted PromotionRolloutPhase = "Aborted"
)

// PromotionRolloutPhase is a high-level summary of a PromotionRollout's
// lifecycle.
type PromotionRolloutPhase string

// IsTerminal returns true if the PromotionRolloutPhase is a terminal one.
func (p PromotionRolloutPhase) IsTerminal() bool {
	switch p {
	case PromotionRolloutPhaseSucceeded,
		PromotionRolloutPhaseFailed,
		PromotionRolloutPhaseErrored,
		PromotionRolloutPhaseAborted:
		return true
	default:
		return false
	}
}

// PromotionRollout expresses the intent to promote a piece of Freight to the
// Targets a Stage governed when the rollout was created. Its Stage, Freight
// and Targets are fixed for its lifetime, giving it a single freight
// transition to represent from start to terminal state.
type PromotionRollout struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Number is the rollout's position among its Stage's rollouts, counting
	// from 1. It is assigned when the rollout is created and never changes.
	Number int64 `json:"number,omitempty"`

	// Spec describes the Stage, the Freight, and the Targets of the
	// PromotionRollout.
	Spec PromotionRolloutSpec `json:"spec"`

	// Status describes the per-Target progress and the aggregate state of the
	// PromotionRollout's Promotions.
	Status PromotionRolloutStatus `json:"status,omitempty"`
}

// GetStatus returns the PromotionRollout's status.
func (p *PromotionRollout) GetStatus() *PromotionRolloutStatus {
	return &p.Status
}

// PromotionRolloutSpec describes the Stage, the Freight, and the Targets of a
// PromotionRollout.
type PromotionRolloutSpec struct {
	// Stage specifies the name of the Stage that promotes the Freight. The
	// Stage is in the same Project as the PromotionRollout.
	Stage string `json:"stage"`

	// Freight specifies the piece of Freight promoted by the Stage. The
	// Freight is in the same Project as the PromotionRollout.
	Freight string `json:"freight"`

	// Targets names the Targets to which this PromotionRollout promotes
	// Freight. Each Target is in the same Project as the PromotionRollout.
	//
	// This is a resolved list, not a selector: the Stage's target selectors
	// are evaluated once, when the rollout is created, and the result
	// recorded here. The list may be empty, which records that the governing
	// Stage governed no Targets at that moment.
	Targets []PromotionRolloutTarget `json:"targets"`

	// UpdateStrategy configures the pace of updating the Targets.
	UpdateStrategy PromotionRolloutUpdateStrategy `json:"updateStrategy,omitempty"`
}

// PromotionRolloutUpdateStrategy is a Stage's TargetUpdateStrategy as a
// PromotionRollout records it when it is created: as numbers of Targets,
// with any percentage the Stage gives resolved against the Stage's Targets,
// as for a PromotionRequest.
type PromotionRolloutUpdateStrategy struct {
	// MaxConcurrent specifies a number of Targets which can be promoted to at
	// the same time. Zero means there is no limit (default).
	//
	// +optional
	MaxConcurrent int64 `json:"maxConcurrent,omitempty"`
	// MaxFailures specifies a number of Target promotions which can fail
	// before the PromotionRollout is considered failed: N failing promotions
	// with maxFailures: N is a success, N+1 is a failure. Defaults to 0,
	// meaning no failures tolerated.
	//
	// +optional
	MaxFailures int64 `json:"maxFailures,omitempty"`
}

// PromotionRolloutTarget names a Target that a PromotionRollout promotes
// Freight to.
type PromotionRolloutTarget struct {
	// Name is the name of the Target.
	Name string `json:"name"`
}

// PromotionRolloutStatus describes the observed aggregate state of a
// PromotionRollout's Promotions.
type PromotionRolloutStatus struct {
	// Conditions contains the last observations of the PromotionRollout's
	// current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase is a high-level summary of the PromotionRollout's lifecycle.
	Phase PromotionRolloutPhase `json:"phase,omitempty"`

	// Message is a display message explaining the current Phase: what the
	// PromotionRollout is waiting on, how far its fan-out has progressed, or
	// why it did not succeed.
	Message string `json:"message,omitempty"`

	// Targets records progress against spec.targets: one entry per Target
	// that has been promoted to, naming the latest Promotion promoting the
	// Freight to it and that Promotion's phase. A Target promoted to more
	// than once within the rollout, for instance after a retry, reports its
	// newest Promotion.
	Targets []PromotionRolloutTargetStatus `json:"targets,omitempty"`

	// Summary tallies the entries of Targets by phase.
	Summary *PromotionRolloutSummary `json:"summary,omitempty"`

	// StartedAt is the time at which the PromotionRollout started.
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// FinishedAt is the time at which the PromotionRollout completed.
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
}

// PromotionRolloutTargetStatus records the state of a single Target selected
// by a PromotionRollout.
type PromotionRolloutTargetStatus struct {
	// Name is the name of the Target.
	Name string `json:"name"`

	// Promotion is the name of the newest Promotion promoting the Freight to
	// this Target.
	Promotion string `json:"promotion,omitempty"`

	// Phase is the phase of that Promotion.
	Phase PromotionPhase `json:"phase,omitempty"`
}

// PromotionRolloutSummary tallies a PromotionRollout's Targets by the phase
// of the newest Promotion to each.
type PromotionRolloutSummary struct {
	// Pending is the number of Targets whose Promotion is Pending.
	Pending int32 `json:"pending,omitempty"`
	// Running is the number of Targets whose Promotion is Running.
	Running int32 `json:"running,omitempty"`
	// Succeeded is the number of Targets whose Promotion Succeeded.
	Succeeded int32 `json:"succeeded,omitempty"`
	// Failed is the number of Targets whose Promotion Failed.
	Failed int32 `json:"failed,omitempty"`
	// Errored is the number of Targets whose Promotion Errored.
	Errored int32 `json:"errored,omitempty"`
	// Aborted is the number of Targets whose Promotion was Aborted.
	Aborted int32 `json:"aborted,omitempty"`
}

// GetConditions returns the PromotionRollout status conditions.
func (s *PromotionRolloutStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}

// SetConditions sets the PromotionRollout status conditions.
func (s *PromotionRolloutStatus) SetConditions(conditions []metav1.Condition) {
	s.Conditions = conditions
}

// PromotionRolloutList contains a list of PromotionRollouts.
type PromotionRolloutList struct {
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromotionRollout `json:"items"`
}
