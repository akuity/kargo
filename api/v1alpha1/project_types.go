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
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes a Project.
	//
	// Deprecated: Create a ProjectConfig resource with the same name as the
	// Project resource in the Project's namespace. The ProjectConfig resource
	// can be used to configure the Project.
	Spec *ProjectSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the Project's current status.
	Status ProjectStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// ProjectSpec is a deprecated alias for ProjectConfigSpec. It is retained for
// backwards compatibility.
//
// Deprecated: Use ProjectConfigSpec instead.
type ProjectSpec = ProjectConfigSpec

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
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge" protobuf:"bytes,3,rep,name=conditions"`
	// Stats contains a summary of the collective state of a Project's
	// constituent resources.
	Stats *ProjectStats `json:"stats,omitempty" protobuf:"bytes,4,opt,name=stats"`
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
	Warehouses WarehouseStats `json:"warehouses,omitempty" protobuf:"bytes,1,opt,name=warehouses"`
	// Stages contains a summary of the collective state of the Project's Stages.
	Stages StageStats `json:"stages,omitempty" protobuf:"bytes,2,opt,name=stages"`
	// Receivers contains a summary of the collective state of the Project's
	// Receivers.
	Receivers ReceiverStats `json:"receivers,omitempty" protobuf:"bytes,3,opt,name=receivers"`
}

// WarehouseStats contains a summary of the collective state of a Project's
// Warehouses.
type WarehouseStats struct {
	// Count contains the total number of Warehouses in the Project.
	Count int64 `json:"count,omitempty" protobuf:"varint,2,opt,name=count"`
	// Health contains a summary of the collective health of a Project's
	// Warehouses.
	Health HealthStats `json:"health,omitempty" protobuf:"bytes,1,opt,name=health"`
}

// StageStats contains a summary of the collective state of a Project's
// Stages.
type StageStats struct {
	// Count contains the total number of Stages in the Project.
	Count int64 `json:"count,omitempty" protobuf:"varint,2,opt,name=count"`
	// Health contains a summary of the collective health of a Project's Stages.
	Health HealthStats `json:"health,omitempty" protobuf:"bytes,1,opt,name=health"`
}

type ReceiverStats struct {
	// Count contains the total number of Receivers in the Project.
	Count int64 `json:"count,omitempty" protobuf:"varint,2,opt,name=count"`
	// Health contains a summary of the collective health of a Project's
	// Receivers.
	Health HealthStats `json:"health,omitempty" protobuf:"bytes,1,opt,name=health"`
}

// HealthStats contains a summary of the collective health of some resource
// type.
type HealthStats struct {
	// Healthy contains the number of resources that are explicitly healthy.
	Healthy int64 `json:"healthy,omitempty" protobuf:"varint,1,opt,name=healthy"`
}

// +kubebuilder:object:root=true

// ProjectList is a list of Project resources.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Project `json:"items" protobuf:"bytes,2,rep,name=items"`
}
