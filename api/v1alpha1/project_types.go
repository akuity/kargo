package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Project is a resource type that reconciles to a specially labeled namespace
// and other TODO: TBD project-level resources.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status describes the Project's current status.
	Status ProjectStatus `json:"status,omitempty"`
}

func (p *Project) GetStatus() *ProjectStatus {
	return &p.Status
}

// ProjectStatus describes a Project's current status.
type ProjectStatus struct {
	// Conditions contains the last observations of the Project's current
	// state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge"`
	// Stats contains a summary of the collective state of a Project's
	// constituent resources.
	Stats *ProjectStats `json:"stats,omitempty"`
}

// GetConditions implements the conditions.Getter interface.
func (w *ProjectStatus) GetConditions() []metav1.Condition {
	return w.Conditions
}

// SetConditions implements the conditions.Setter interface.
func (w *ProjectStatus) SetConditions(conditions []metav1.Condition) {
	w.Conditions = conditions
}

// ProjectStats contains a summary of the collective state of a Project's
// constituent resources.
type ProjectStats struct {
	// Warehouses contains a summary of the collective state of the Project's
	// Warehouses.
	Warehouses WarehouseStats `json:"warehouses,omitempty"`
	// Stages contains a summary of the collective state of the Project's Stages.
	Stages StageStats `json:"stages,omitempty"`
	// Targets contains a summary of the Project's Targets and of how the latest
	// round of promotion to them fared. It is absent for a Project with no
	// Targets and no target-aware Stages.
	//
	// +optional
	Targets *TargetStats `json:"targets,omitempty"`
}

// TargetStats summarizes a Project's Targets and how the latest round of
// promotion to each of them turned out.
//
// A Target is governed by every target-aware Stage whose selectors match it,
// and each such Stage promotes to it separately. Count counts distinct
// Targets, so a Target governed by two Stages counts once there. Promotion
// counts Stage and Target pairs, so that same Target counts twice there, once
// for each Stage that promoted to it. A progress bar should therefore use the
// total of Promotion as its denominator, not Count.
type TargetStats struct {
	// Count is the number of distinct Targets in the Project.
	Count int64 `json:"count,omitempty"`
	// Promotion tallies, by Promotion phase, every Stage and Target pair in the
	// Project's latest rounds of promotion. Each target-aware Stage contributes
	// one entry per Target named by its latest PromotionRequest -- the request
	// the Stage reports as current, else as last -- in the phase of the child
	// Promotion promoting to that Target. It is the sum of those requests'
	// summaries and so shares their type.
	//
	// A request that has not yet produced child Promotions contributes all of
	// its Targets as Pending. A request that reached a terminal phase without
	// ever producing children contributes all of its Targets in that phase.
	Promotion PromotionRequestSummary `json:"promotion,omitempty"`
	// Unknown is the number of target-aware Stages whose latest
	// PromotionRequest no longer exists, typically because it was garbage
	// collected. Nothing can be said about the outcome of promoting from such a
	// Stage, so it is left out of Promotion and counted here instead.
	Unknown int64 `json:"unknown,omitempty"`
}

// WarehouseStats contains a summary of the collective state of a Project's
// Warehouses.
type WarehouseStats struct {
	// Count contains the total number of Warehouses in the Project.
	Count int64 `json:"count,omitempty"`
	// Health contains a summary of the collective health of a Project's
	// Warehouses.
	Health HealthStats `json:"health,omitempty"`
}

// StageStats contains a summary of the collective state of a Project's
// Stages.
type StageStats struct {
	// Count contains the total number of Stages in the Project.
	Count int64 `json:"count,omitempty"`
	// Health contains a summary of the collective health of a Project's Stages.
	Health HealthStats `json:"health,omitempty"`
}

// HealthStats contains a summary of the collective health of some resource
// type.
type HealthStats struct {
	// Healthy contains the number of resources that are explicitly healthy.
	Healthy int64 `json:"healthy,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList is a list of Project resources.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}
