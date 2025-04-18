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
	Spec *ProjectSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the Project's current status.
	Status ProjectStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

func (p *Project) GetStatus() *ProjectStatus {
	return &p.Status
}

// ProjectSpec describes a Project.
type ProjectSpec struct {
	// PromotionPolicies defines policies governing the promotion of Freight to
	// specific Stages within this Project.
	PromotionPolicies []PromotionPolicy `json:"promotionPolicies,omitempty" protobuf:"bytes,1,rep,name=promotionPolicies"`
}

// PromotionPolicy defines policies governing the promotion of Freight to a
// specific Stage.
type PromotionPolicy struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Stage string `json:"stage" protobuf:"bytes,1,opt,name=stage"`
	// AutoPromotionEnabled indicates whether new Freight can automatically be
	// promoted into the Stage referenced by the Stage field. Note: There are may
	// be other conditions also required for an auto-promotion to occur. This
	// field defaults to false, but is commonly set to true for Stages that
	// subscribe to Warehouses instead of other, upstream Stages. This allows
	// users to define Stages that are automatically updated as soon as new
	// artifacts are detected.
	AutoPromotionEnabled bool `json:"autoPromotionEnabled,omitempty" protobuf:"varint,2,opt,name=autoPromotionEnabled"`
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
	// Phase describes the Project's current phase.
	//
	// Deprecated: Use the Conditions field instead.
	Phase ProjectPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`
	// Message is a display message about the Project, including any errors
	// preventing the Project from being reconciled. i.e. If the Phase field has a
	// value of CreationFailed, this field can be expected to explain why.
	//
	// Deprecated: Use the Conditions field instead.
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`

	Stats ProjectStats `json:"stats,omitempty" protobuf:"bytes,4,opt,name=stats"`
}

func (w *ProjectStatus) GetConditions() []metav1.Condition {
	return w.Conditions
}

func (w *ProjectStatus) SetConditions(conditions []metav1.Condition) {
	w.Conditions = conditions
}

type ProjectStats struct {
	Warehouses WarehouseStats `json:"warehouses,omitempty" protobuf:"bytes,1,opt,name=warehouses"`
	Stages     StageStats     `json:"stages,omitempty" protobuf:"bytes,2,opt,name=stages"`
}

type WarehouseStats struct {
	Health HealthStats `json:"health,omitempty" protobuf:"bytes,1,opt,name=health"`
}

type StageStats struct {
	Health HealthStats `json:"health,omitempty" protobuf:"bytes,1,opt,name=health"`
}

type HealthStats struct {
	Healthy   int64 `json:"healthy,omitempty" protobuf:"varint,1,opt,name=healthy"`
	Unhealthy int64 `json:"unhealthy,omitempty" protobuf:"varint,2,opt,name=unhealthy"`
	Unknown   int64 `json:"unknown,omitempty" protobuf:"varint,3,opt,name=unknown"`
}

// +kubebuilder:object:root=true

// ProjectList is a list of Project resources.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Project `json:"items" protobuf:"bytes,2,rep,name=items"`
}
