package v1alpha1

import (
	"time"

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
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Stage string `json:"stage" protobuf:"bytes,1,opt,name=stage"`
	// Freight specifies the piece of Freight to be promoted into the Stage
	// referenced by the Stage field.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Freight string `json:"freight" protobuf:"bytes,2,opt,name=freight"`
	// Vars is a list of variables that can be referenced by expressions in
	// promotion steps.
	Vars []PromotionVariable `json:"vars,omitempty" protobuf:"bytes,4,rep,name=vars"`
	// Steps specifies the directives to be executed as part of this Promotion.
	// The order in which the directives are executed is the order in which they
	// are listed in this field.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:XValidation:message="Promotion step must have uses set and must not reference a task",rule="has(self.uses) && !has(self.task)"
	Steps []PromotionStep `json:"steps" protobuf:"bytes,3,rep,name=steps"`
}

// PromotionVariable describes a single variable that may be referenced by
// expressions in promotion steps.
type PromotionVariable struct {
	// Name is the name of the variable.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[a-zA-Z_]\w*$
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Value is the value of the variable. It is allowed to utilize expressions
	// in the value.
	// See https://docs.kargo.io/references/expression-language for details.
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

// PromotionTaskReference describes a reference to a PromotionTask.
type PromotionTaskReference struct {
	// Name is the name of the (Cluster)PromotionTask.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Kind is the type of the PromotionTask. Can be either PromotionTask or
	// ClusterPromotionTask, default is PromotionTask.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=PromotionTask;ClusterPromotionTask
	Kind string `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
}

// PromotionStepRetry describes the retry policy for a PromotionStep.
type PromotionStepRetry struct {
	// Timeout is the soft maximum interval in which a step that returns a Running
	// status (which typically indicates it's waiting for something to happen)
	// may be retried.
	//
	// The maximum is a soft one because the check for whether the interval has
	// elapsed occurs AFTER the step has run. This effectively means a step may
	// run ONCE beyond the close of the interval.
	//
	// If this field is set to nil, the effective default will be a step-specific
	// one. If no step-specific default exists (i.e. is also nil), the effective
	// default will be the system-wide default of 0.
	//
	// A value of 0 will cause the step to be retried indefinitely unless the
	// ErrorThreshold is reached.
	Timeout *metav1.Duration `json:"timeout,omitempty" protobuf:"bytes,1,opt,name=timeout"`
	// ErrorThreshold is the number of consecutive times the step must fail (for
	// any reason) before retries are abandoned and the entire Promotion is marked
	// as failed.
	//
	// If this field is set to 0, the effective default will be a step-specific
	// one. If no step-specific default exists (i.e. is also 0), the effective
	// default will be the system-wide default of 1.
	//
	// A value of 1 will cause the Promotion to be marked as failed after just
	// a single failure; i.e. no retries will be attempted.
	//
	// There is no option to specify an infinite number of retries using a value
	// such as -1.
	//
	// In a future release, Kargo is likely to become capable of distinguishing
	// between recoverable and non-recoverable step failures. At that time, it is
	// planned that unrecoverable failures will not be subject to this threshold
	// and will immediately cause the Promotion to be marked as failed without
	// further condition.
	ErrorThreshold uint32 `json:"errorThreshold,omitempty" protobuf:"varint,2,opt,name=errorThreshold"`
}

// GetTimeout returns the Timeout field with the given fallback value.
func (r *PromotionStepRetry) GetTimeout(fallback *time.Duration) *time.Duration {
	if r == nil || r.Timeout == nil {
		return fallback
	}
	return &r.Timeout.Duration
}

// GetErrorThreshold returns the ErrorThreshold field with the given fallback
// value.
func (r *PromotionStepRetry) GetErrorThreshold(fallback uint32) uint32 {
	if r == nil || r.ErrorThreshold == 0 {
		return fallback
	}
	return r.ErrorThreshold
}

// PromotionStep describes a directive to be executed as part of a Promotion.
//
// +kubebuilder:validation:XValidation:message="inputs must not be set when task is set",rule="!(has(self.task) && self.inputs.size() > 0)"
type PromotionStep struct {
	// Uses identifies a runner that can execute this step.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Uses string `json:"uses,omitempty" protobuf:"bytes,1,opt,name=uses"`
	// Task is a reference to a PromotionTask that should be deflated into a
	// Promotion when it is built from a PromotionTemplate.
	Task *PromotionTaskReference `json:"task,omitempty" protobuf:"bytes,5,opt,name=task"`
	// As is the alias this step can be referred to as.
	As string `json:"as,omitempty" protobuf:"bytes,2,opt,name=as"`
	// Retry is the retry policy for this step.
	Retry *PromotionStepRetry `json:"retry,omitempty" protobuf:"bytes,4,opt,name=retry"`
	// Inputs is a map of inputs that can used to parameterize the execution
	// of the PromotionStep and can be referenced by expressions in the Config.
	//
	// When a PromotionStep is inflated from a PromotionTask, the inputs
	// specified in the PromotionTask are set based on the inputs specified
	// in the Config of the PromotionStep that references the PromotionTask.
	Inputs map[string]string `json:"inputs,omitempty" protobuf:"bytes,6,rep,name=inputs"`
	// Config is opaque configuration for the PromotionStep that is understood
	// only by each PromotionStep's implementation. It is legal to utilize
	// expressions in defining values at any level of this block.
	// See https://docs.kargo.io/references/expression-language for details.
	Config *apiextensionsv1.JSON `json:"config,omitempty" protobuf:"bytes,3,opt,name=config"`
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
	// StepExecutionMetadata tracks metadata pertaining to the execution
	// of individual promotion steps.
	StepExecutionMetadata StepExecutionMetadataList `json:"stepExecutionMetadata,omitempty" protobuf:"bytes,11,rep,name=stepExecutionMetadata"`
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
func (s *PromotionStatus) WithPhase(phase PromotionPhase) *PromotionStatus {
	status := s.DeepCopy()
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

// StepExecutionMetadataList is a list of StepExecutionMetadata.
type StepExecutionMetadataList []StepExecutionMetadata

// StepExecutionMetadata tracks metadata pertaining to the execution of
// a promotion step.
type StepExecutionMetadata struct {
	// Alias is the alias of the step.
	Alias string `json:"alias,omitempty" protobuf:"bytes,1,opt,name=alias"`
	// StartedAt is the time at which the first attempt to execute the step
	// began.
	StartedAt *metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`
	// FinishedAt is the time at which the final attempt to execute the step
	// completed.
	FinishedAt *metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,3,opt,name=finishedAt"`
	// ErrorCount tracks consecutive failed attempts to execute the step.
	ErrorCount uint32 `json:"errorCount,omitempty" protobuf:"varint,4,opt,name=errorCount"`
	// Status is the high-level outcome of the step.
	Status PromotionPhase `json:"status,omitempty" protobuf:"bytes,5,opt,name=status"`
	// Message is a display message about the step, including any errors.
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}
