package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	WebhookReceivers []WebhookReceiverConfig `json:"webhookReceivers,omitempty" protobuf:"bytes,2,rep,name=webhookReceivers"`
}

// ProjectConfigStatus describes the current status of a ProjectConfig.
type ProjectConfigStatus struct {
	// Conditions contains the last observations of the Project Config's current
	// state.
	//
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
	// ObservedGeneration represents the .metadata.generation that this
	// ProjectConfig was reconciled against.
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`
	// LastHandledRefresh holds the value of the most recent AnnotationKeyRefresh
	// annotation that was handled by the controller. This field can be used to
	// determine whether the request to refresh the resource has been handled.
	// +optional
	LastHandledRefresh string `json:"lastHandledRefresh,omitempty" protobuf:"bytes,4,opt,name=lastHandledRefresh"`
	// WebhookReceivers describes the status of Project-specific webhook
	// receivers.
	WebhookReceivers []WebhookReceiverDetails `json:"webhookReceivers,omitempty" protobuf:"bytes,2,rep,name=webhookReceivers"`
}

// GetConditions implements the conditions.Getter interface.
func (p *ProjectConfigStatus) GetConditions() []metav1.Condition {
	return p.Conditions
}

// SetConditions implements the conditions.Setter interface.
func (p *ProjectConfigStatus) SetConditions(conditions []metav1.Condition) {
	p.Conditions = conditions
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
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Bitbucket contains the configuration for a webhook receiver that is
	// compatible with Bitbucket payloads.
	Bitbucket *BitbucketWebhookReceiverConfig `json:"bitbucket,omitempty" protobuf:"bytes,5,opt,name=bitbucket"`
	// DockerHub contains the configuration for a webhook receiver that is
	// compatible with DockerHub payloads.
	DockerHub *DockerHubWebhookReceiverConfig `json:"dockerhub,omitempty" protobuf:"bytes,6,opt,name=dockerhub"`
	// GitHub contains the configuration for a webhook receiver that is compatible
	// with GitHub payloads.
	GitHub *GitHubWebhookReceiverConfig `json:"github,omitempty" protobuf:"bytes,2,opt,name=github"`
	// GitLab contains the configuration for a webhook receiver that is compatible
	// with GitLab payloads.
	GitLab *GitLabWebhookReceiverConfig `json:"gitlab,omitempty" protobuf:"bytes,3,opt,name=gitlab"`
	// Quay contains the configuration for a webhook receiver that is compatible
	// with Quay payloads.
	Quay *QuayWebhookReceiverConfig `json:"quay,omitempty" protobuf:"bytes,4,opt,name=quay"`
	// Azure contains the configuration for a webhook receiver that is compatible
	// with Azure Container Registry (ACR) and Azure DevOps payloads.
	Azure *AzureWebhookReceiverConfig `json:"azure,omitempty" protobuf:"bytes,8,opt,name=azure"`
	// Gitea contains the configuration for a webhook receiver that is compatible
	// with Gitea payloads.
	Gitea *GiteaWebhookReceiverConfig `json:"gitea,omitempty" protobuf:"bytes,7,opt,name=gitea"`
}

// GiteaWebhookReceiverConfig describes a webhook receiver that is compatible
// with Gitea payloads.
type GiteaWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "cluster Secrets" namespace.
	//
	// The Secret's data map is expected to contain a `secret` key whose value is
	// the shared secret used to authenticate the webhook requests sent by Gitea.
	// For more information please refer to the Gitea documentation:
	//   https://docs.gitea.io/en-us/webhooks/
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// BitbucketWebhookReceiverConfig describes a webhook receiver that is
// compatible with Bitbucket payloads.
type BitbucketWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "cluster Secrets" namespace.
	//
	// The Secret's data map is expected to contain a `secret` key whose
	// value is the shared secret used to authenticate the webhook requests sent
	// by Bitbucket. For more information please refer to the Bitbucket
	// documentation:
	//   https://support.atlassian.com/bitbucket-cloud/docs/manage-webhooks/
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// DockerHubWebhookReceiverConfig describes a webhook receiver that is
// compatible with Docker Hub payloads.
type DockerHubWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// The Secret's data map is expected to contain a `secret` key whose value
	// does NOT need to be shared directly with Docker Hub when registering a
	// webhook. It is used only by Kargo to create a complex, hard-to-guess URL,
	// which implicitly serves as a shared secret. For more information about
	// Docker Hub webhooks, please refer to the Docker documentation:
	//   https://docs.docker.com/docker-hub/webhooks/
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// GitHubWebhookReceiverConfig describes a webhook receiver that is compatible
// with GitHub payloads.
type GitHubWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "cluster Secrets" namespace.
	//
	// The Secret's data map is expected to contain a `secret` key whose value is
	// the shared secret used to authenticate the webhook requests sent by GitHub.
	// For more information please refer to GitHub documentation:
	//   https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// GitLabWebhookReceiverConfig describes a webhook receiver that is compatible
// with GitLab payloads.
type GitLabWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "cluster Secrets" namespace.
	//
	// The secret is expected to contain a `secret-token` key containing the
	// shared secret specified when registering the webhook in GitLab. For more
	// information about this token, please refer to the GitLab documentation:
	//   https://docs.gitlab.com/user/project/integrations/webhooks/
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// QuayWebhookReceiverConfig describes a webhook receiver that is compatible
// with Quay.io payloads.
type QuayWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "cluster Secrets" namespace.
	//
	// The Secret's data map is expected to contain a `secret` key whose value
	// does NOT need to be shared directly with Quay when registering a
	// webhook. It is used only by Kargo to create a complex, hard-to-guess URL,
	// which implicitly serves as a shared secret. For more information about
	// Quay webhooks, please refer to the Quay documentation:
	//   https://docs.quay.io/guides/notifications.html
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// AzureWebhookReceiverConfig describes a webhook receiver that is compatible
// with Azure Container Registry (ACR) and Azure DevOps payloads.
type AzureWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "cluster Secrets" namespace.
	//
	// The Secret's data map is expected to contain a `secret` key whose value
	// does NOT need to be shared directly with Azure when registering a webhook.
	// It is used only by Kargo to create a complex, hard-to-guess URL,
	// which implicitly serves as a shared secret. For more information about
	// Azure webhooks, please refer to the Azure documentation:
	//
	//  Azure Container Registry:
	//	https://learn.microsoft.com/en-us/azure/container-registry/container-registry-repositories
	//
	//  Azure DevOps:
	//	http://learn.microsoft.com/en-us/azure/devops/service-hooks/services/webhooks?view=azure-devops
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// WebhookReceiverDetails encapsulates the details of a webhook receiver.
type WebhookReceiverDetails struct {
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
