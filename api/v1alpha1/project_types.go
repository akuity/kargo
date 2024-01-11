package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Project is a resource type that reconciles to a specially labeled namespace
// and other TODO: TBD project-level resources.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes a Project.
	//
	//+kubebuilder:validation:Required
	Spec *ProjectSpec `json:"spec"`
	// Status describes the Project's current status.
	Status ProjectStatus `json:"status,omitempty"`
}

func (p *Project) GetStatus() *ProjectStatus {
	return &p.Status
}

// ProjectSpec describes a Project.
type ProjectSpec struct {
	// TODO: Figure out the attributes of a ProjectSpec.
}

// ProjectStatus describes a Project's current status.
type ProjectStatus struct {
	// TODO: Figure out the attributes of a ProjectStatus.
}

//+kubebuilder:object:root=true

// ProjectList is a list of Project resources.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}
