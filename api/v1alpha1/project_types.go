package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProjectPhase string

const (
	// ProjectPhaseInitializing denotes a Project that is not yet fully
	// initialized.
	ProjectPhaseInitializing ProjectPhase = "Initializing"
	// ProjectPhaseInitializationFailed denotes a Project while failed to
	// initialize properly.
	ProjectPhaseInitializationFailed ProjectPhase = "InitializationFailed"
	// ProjectPhaseReady denotes a Project that is fully initialized.
	ProjectPhaseReady ProjectPhase = "Ready"
)

// IsTerminal returns true if the ProjectPhase is a terminal one.
func (p *ProjectPhase) IsTerminal() bool {
	switch *p {
	case ProjectPhaseInitializationFailed, ProjectPhaseReady:
		return true
	default:
		return false
	}
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Project is a resource type that reconciles to a specially labeled namespace
// and other TODO: TBD project-level resources.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes a Project.
	Spec *ProjectSpec `json:"spec,omitempty"`
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
	// Phase describes the Project's current phase.
	Phase ProjectPhase `json:"phase,omitempty"`
	// Message is a display message about the Project, including any errors
	// preventing the Project from being reconciled. i.e. If the Phase field has a
	// value of CreationFailed, this field can be expected to explain why.
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectList is a list of Project resources.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}
