package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WebhookReceiverTypeGitHub = "GitHub"
	// TODO(fuskovic): Add more receiver enum types(e.g. Dockerhub, Quay, Gitlab, etc...)
)

const (
	WebhookReceiverSecretKeyGithub = "token"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// ProjectConfig is a resource type that describes the configuration of a
// Project.
type ProjectConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes the configuration of a Project.
	Spec ProjectConfigSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the current status of a ProjectConfig.
	Status ProjectConfigStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

func (p *ProjectConfig) GetStatus() *ProjectConfigStatus {
	return &p.Status
}

// ProjectConfigSpec describes the configuration of a Project.
type ProjectConfigSpec struct {
	// PromotionPolicies defines policies governing the promotion of Freight to
	// specific Stages within the Project.
	PromotionPolicies []PromotionPolicy `json:"promotionPolicies,omitempty" protobuf:"bytes,1,rep,name=promotionPolicies"`
	// WebhookReceivers describes Project-specific webhook receivers used for
	// processing events from various external platforms
	//
	// +kubebuilder:validation:MaxItems=5
	// +kubebuilder:validation:XValidation:message="WebhookReceiverConfig must have a unique name",rule="self.all(i, size(self.filter(j, i.name == j.name)) == 1)"
	WebhookReceivers []WebhookReceiverConfig `json:"webhookReceivers,omitempty" protobuf:"bytes,2,rep,name=receivers"`
}

// ProjectConfigStatus describes the current status of a ProjectConfig.
type ProjectConfigStatus struct {
	// Conditions contains the last observations of the Project Config's current state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
	// WebhookReceivers describes the status of Project-specific webhook receivers.
	WebhookReceivers []WebhookReceiver `json:"webhookReceivers,omitempty" protobuf:"bytes,2,rep,name=receivers"`
}

// GetConditions implements the conditions.Getter interface.
func (w *ProjectConfigStatus) GetConditions() []metav1.Condition {
	return w.Conditions
}

// SetConditions implements the conditions.Setter interface.
func (w *ProjectConfigStatus) SetConditions(conditions []metav1.Condition) {
	w.Conditions = conditions
}

// PromotionPolicy defines policies governing the promotion of Freight to a
// specific Stage.
//
// +kubebuilder:validation:XValidation:message="PromotionPolicy must have exactly one of stage or stageSelector set",rule="has(self.stage) ? !has(self.stageSelector) : has(self.stageSelector)"
type PromotionPolicy struct {
	// Stage is the name of the Stage to which this policy applies.
	//
	// Deprecated: Use StageSelector instead.
	//
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Stage string `json:"stage,omitempty" protobuf:"bytes,1,opt,name=stage"`
	// StageSelector is a selector that matches the Stage resource to which
	// this policy applies.
	StageSelector *PromotionPolicySelector `json:"stageSelector,omitempty" protobuf:"bytes,3,opt,name=stageSelector"`
	// AutoPromotionEnabled indicates whether new Freight can automatically be
	// promoted into the Stage referenced by the Stage field. Note: There are may
	// be other conditions also required for an auto-promotion to occur. This
	// field defaults to false, but is commonly set to true for Stages that
	// subscribe to Warehouses instead of other, upstream Stages. This allows
	// users to define Stages that are automatically updated as soon as new
	// artifacts are detected.
	AutoPromotionEnabled bool `json:"autoPromotionEnabled,omitempty" protobuf:"varint,2,opt,name=autoPromotionEnabled"`
}

// WebhookReceiverConfig describes the configuration for a single webhook
// receiver.
type WebhookReceiverConfig struct {
	// Name is the name of the webhook receiver.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// GitHub contains the configuration for a webhook receiver that is compatible with
	// GitHub payloads.
	//
	// TODO(fuskovic): Make this mutually exclusive with configs for other platforms.
	//
	// +kubebuilder:validation:Required
	GitHub *GitHubWebhookReceiver `json:"github,omitempty" protobuf:"bytes,2,opt,name=github"`
}

// GitHubWebhookReceiver describes a webhook receiver that is compatible with
// GitHub payloads.
type GitHubWebhookReceiver struct {
	// SecretRef contains a reference to a Secret in the same namespace as the ProjectConfig.
	//
	// The Secret is expected to contain a `token` key with the secret token configured for
	// in GitHub for the webhook. For more information about this token, please refer to the
	// GitHub documentation: https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
	//
	// The value of the token key goes in the "Secret" field when registering a GitHub App or webhook in the GitHub UI.
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// WebhookReceiver describes a path used to receive webhook events.
type WebhookReceiver struct {
	// Name is the name of the webhook receiver.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Path is the path to the receiver's webhook endpoint.
	Path string `json:"path,omitempty" protobuf:"bytes,3,opt,name=path"`
	// URL includes the full address of the receiver's webhook endpoint.
	URL string `json:"url,omitempty" protobuf:"bytes,4,opt,name=url"`
}

// PromotionPolicySelector is a selector that matches the resource to which
// this policy applies. It can be used to match a specific resource by name or
// to match a set of resources by label.
type PromotionPolicySelector struct {
	// Name is the name of the resource to which this policy applies.
	//
	// It can be an exact name, a regex pattern (with prefix "regex:"), or a
	// glob pattern (with prefix "glob:").
	//
	// When both Name and LabelSelector are specified, the Name is ANDed with
	// the LabelSelector. I.e., the resource must match both the Name and
	// LabelSelector to be selected by this policy.
	//
	// NOTE: Using a specific exact name is the most secure option. Pattern
	// matching via regex or glob can be exploited by users with permissions to
	// match promotion policies that weren't intended to apply to their
	// resources. For example, a user could create a resource with a name
	// deliberately crafted to match the pattern, potentially bypassing intended
	// promotion controls.
	//
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// LabelSelector is a selector that matches the resource to which this policy
	// applies.
	//
	// When both Name and LabelSelector are specified, the Name is ANDed with
	// the LabelSelector. I.e., the resource must match both the Name and
	// LabelSelector to be selected by this policy.
	//
	// NOTE: Using label selectors introduces security risks as users with
	// appropriate permissions could create new resources with labels that match
	// the selector, potentially enabling unauthorized auto-promotion.
	// For sensitive environments, exact Name matching provides tighter control.
	*metav1.LabelSelector `json:",inline" protobuf:"bytes,2,opt,name=labelSelector"`
}

// +kubebuilder:object:root=true

// ProjectConfigList is a list of ProjectConfig resources.
type ProjectConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []ProjectConfig `json:"items" protobuf:"bytes,2,rep,name=items"`
}
