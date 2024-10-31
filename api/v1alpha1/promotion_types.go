package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type PromotionPhase string

const (
	// PromotionPhasePending denotes a Promotion that has not been executed yet.
	// i.e. It is currently waiting in a queue. Queues are stage-specific and
	// prioritized by Promotion creation time.
	PromotionPhasePending PromotionPhase = "Pending"
	// PromotionPhaseRunning denotes a Promotion that is actively being executed.
	//
	// TODO: "Active" is the operative word here. We are leaving room for the
	// possibility in the near future that an in-progress Promotion might be
	// paused/suspended pending some user action.
	PromotionPhaseRunning PromotionPhase = "Running"
	// PromotionPhaseSucceeded denotes a Promotion that has been successfully
	// executed.
	PromotionPhaseSucceeded PromotionPhase = "Succeeded"
	// PromotionPhaseFailed denotes a Promotion that has failed
	PromotionPhaseFailed PromotionPhase = "Failed"
	// PromotionPhaseErrored denotes a Promotion that has failed for technical
	// reasons. Further information about the failure can be found in the
	// Promotion's status.
	PromotionPhaseErrored PromotionPhase = "Errored"
	// PromotionPhaseAborted denotes a Promotion that has been aborted by a
	// user.
	PromotionPhaseAborted PromotionPhase = "Aborted"
)

// IsTerminal returns true if the PromotionPhase is a terminal one.
func (p *PromotionPhase) IsTerminal() bool {
	switch *p {
	case PromotionPhaseSucceeded, PromotionPhaseFailed, PromotionPhaseErrored, PromotionPhaseAborted:
		return true
	default:
		return false
	}
}

// +kubebuilder:resource:shortName={promo,promos}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Shard,type=string,JSONPath=`.metadata.labels.kargo\.akuity\.io/shard`
// +kubebuilder:printcolumn:name=Stage,type=string,JSONPath=`.spec.stage`
// +kubebuilder:printcolumn:name=Freight,type=string,JSONPath=`.spec.freight`
// +kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Promotion represents a request to transition a particular Stage into a
// particular Freight.
type Promotion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes the desired transition of a specific Stage into a specific
	// Freight.
	//
	// +kubebuilder:validation:Required
	Spec PromotionSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the current state of the transition represented by this
	// Promotion.
	Status PromotionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

func (p *Promotion) GetStatus() *PromotionStatus {
	return &p.Status
}

// PromotionSpec describes the desired transition of a specific Stage into a
// specific Freight.
type PromotionSpec struct {
	// Stage specifies the name of the Stage to which this Promotion
	// applies. The Stage referenced by this field MUST be in the same
	// namespace as the Promotion.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Stage string `json:"stage" protobuf:"bytes,1,opt,name=stage"`
	// Freight specifies the piece of Freight to be promoted into the Stage
	// referenced by the Stage field.
	//
	// +kubebuilder:validation:MinLength=1
	Freight string `json:"freight" protobuf:"bytes,2,opt,name=freight"`
	// Steps specifies the directives to be executed as part of this Promotion.
	// The order in which the directives are executed is the order in which they
	// are listed in this field.
	Steps []PromotionStep `json:"steps,omitempty" protobuf:"bytes,3,rep,name=steps"`
}

// PromotionStep describes a directive to be executed as part of a Promotion.
type PromotionStep struct {
	// Uses identifies a runner that can execute this step.
	//
	// +kubebuilder:validation:MinLength=1
	Uses string `json:"uses" protobuf:"bytes,1,opt,name=uses"`
	// As is the alias this step can be referred to as.
	As string `json:"as,omitempty" protobuf:"bytes,2,opt,name=as"`
	// Config is the configuration for the directive.
	Config *apiextensionsv1.JSON `json:"config,omitempty" protobuf:"bytes,3,opt,name=config"`
}

// GetConfig returns the Config field as unmarshalled YAML.
func (s *PromotionStep) GetConfig() map[string]any {
	if s.Config == nil {
		return nil
	}

	var config map[string]any
	if err := yaml.Unmarshal(s.Config.Raw, &config); err != nil {
		return nil
	}
	return config
}

// PromotionStatus describes the current state of the transition represented by
// a Promotion.
type PromotionStatus struct {
	// LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh
	// annotation that was handled by the controller. This field can be used to
	// determine whether the request to refresh the resource has been handled.
	// +optional
	LastHandledRefresh string `json:"lastHandledRefresh,omitempty" protobuf:"bytes,4,opt,name=lastHandledRefresh"`
	// Phase describes where the Promotion currently is in its lifecycle.
	Phase PromotionPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`
	// Message is a display message about the promotion, including any errors
	// preventing the Promotion controller from executing this Promotion.
	// i.e. If the Phase field has a value of Failed, this field can be expected
	// to explain why.
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	// Freight is the detail of the piece of freight that was referenced by this promotion.
	Freight *FreightReference `json:"freight,omitempty" protobuf:"bytes,5,opt,name=freight"`
	// FreightCollection contains the details of the piece of Freight referenced
	// by this Promotion as well as any additional Freight that is carried over
	// from the target Stage's current state.
	FreightCollection *FreightCollection `json:"freightCollection,omitempty" protobuf:"bytes,7,opt,name=freightCollection"`
	// HealthChecks contains the health check directives to be executed after
	// the Promotion has completed.
	HealthChecks []HealthCheckStep `json:"healthChecks,omitempty" protobuf:"bytes,8,rep,name=healthChecks"`
	// FinishedAt is the time when the promotion was completed.
	FinishedAt *metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,6,opt,name=finishedAt"`
	// CurrentStep is the index of the current promotion step being executed. This
	// permits steps that have already run successfully to be skipped on
	// subsequent reconciliations attempts.
	CurrentStep int64 `json:"currentStep,omitempty" protobuf:"varint,9,opt,name=currentStep"`
	// State stores the state of the promotion process between reconciliation
	// attempts.
	State *apiextensionsv1.JSON `json:"state,omitempty" protobuf:"bytes,10,opt,name=state"`
}

// GetState returns the State field as unmarshalled YAML.
func (s *PromotionStatus) GetState() map[string]any {
	if s.State == nil {
		return nil
	}

	var state map[string]any
	if err := yaml.Unmarshal(s.State.Raw, &state); err != nil {
		return nil
	}
	return state
}

// HealthCheckStep describes a health check directive which can be executed by
// a Stage to verify the health of a Promotion result.
type HealthCheckStep struct {
	// Uses identifies a runner that can execute this step.
	//
	// +kubebuilder:validation:MinLength=1
	Uses string `json:"uses" protobuf:"bytes,1,opt,name=uses"`

	// Config is the configuration for the directive.
	Config *apiextensionsv1.JSON `json:"config,omitempty" protobuf:"bytes,2,opt,name=config"`
}

// GetConfig returns the Config field as unmarshalled YAML.
func (s *HealthCheckStep) GetConfig() map[string]any {
	if s.Config == nil {
		return nil
	}

	var config map[string]any
	if err := yaml.Unmarshal(s.Config.Raw, &config); err != nil {
		return nil
	}
	return config
}

// WithPhase returns a copy of PromotionStatus with the given phase
func (p *PromotionStatus) WithPhase(phase PromotionPhase) *PromotionStatus {
	status := p.DeepCopy()
	status.Phase = phase
	return status
}

// +kubebuilder:object:root=true

// PromotionList contains a list of Promotion
type PromotionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Promotion `json:"items" protobuf:"bytes,2,rep,name=items"`
}
