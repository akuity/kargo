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
	// Harbor contains the configuration for a webhook receiver that is compatible
	// with Harbor payloads.
	Harbor *HarborWebhookReceiverConfig `json:"harbor,omitempty" protobuf:"bytes,10,opt,name=harbor"`
	// Quay contains the configuration for a webhook receiver that is compatible
	// with Quay payloads.
	Quay *QuayWebhookReceiverConfig `json:"quay,omitempty" protobuf:"bytes,4,opt,name=quay"`
	// Artifactory contains the configuration for a webhook receiver that is
	// compatible with JFrog Artifactory payloads.
	Artifactory *ArtifactoryWebhookReceiverConfig `json:"artifactory,omitempty" protobuf:"bytes,9,opt,name=artifactory"`
	// Azure contains the configuration for a webhook receiver that is compatible
	// with Azure Container Registry (ACR) and Azure DevOps payloads.
	Azure *AzureWebhookReceiverConfig `json:"azure,omitempty" protobuf:"bytes,8,opt,name=azure"`
	// Gitea contains the configuration for a webhook receiver that is compatible
	// with Gitea payloads.
	Gitea *GiteaWebhookReceiverConfig `json:"gitea,omitempty" protobuf:"bytes,7,opt,name=gitea"`
	// Generic contains the configuration for a generic webhook receiver.
	Generic *GenericWebhookReceiverConfig `json:"generic,omitempty" protobuf:"bytes,11,opt,name=generic"`
}

// GiteaWebhookReceiverConfig describes a webhook receiver that is compatible
// with Gitea payloads.
type GiteaWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "system resources" namespace.
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
	// designated "system resources" namespace.
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
	// designated "system resources" namespace.
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
	// designated "system resources" namespace.
	//
	// The secret is expected to contain a `secret-token` key containing the
	// shared secret specified when registering the webhook in GitLab. For more
	// information about this token, please refer to the GitLab documentation:
	//   https://docs.gitlab.com/user/project/integrations/webhooks/
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
}

// HarborWebhookReceiverConfig describes a webhook receiver that is compatible
// with Harbor payloads.
type HarborWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "system resources" namespace.
	//
	// The secret is expected to contain an `auth-header` key containing the "auth
	// header" specified when registering the webhook in Harbor. For more
	// information, please refer to the Harbor documentation:
	//   https://goharbor.io/docs/main/working-with-projects/project-configuration/configure-webhooks/
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
	// designated "system resources" namespace.
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

// ArtifactoryWebhookReceiverConfig describes a webhook receiver that is
// compatible with JFrog Artifactory payloads.
type ArtifactoryWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "system resources" namespace.
	//
	// The Secret's data map is expected to contain a `secret-token` key whose
	// value is the shared secret used to authenticate the webhook requests sent
	// by JFrog Artifactory. For more information please refer to the JFrog
	// Artifactory documentation:
	//   https://jfrog.com/help/r/jfrog-platform-administration-documentation/webhooks
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`
	// VirtualRepoName is the name of an Artifactory virtual repository.
	//
	// When unspecified, the Artifactory webhook receiver depends on the value of
	// the webhook payload's `data.repo_key` field when inferring the URL of the
	// repository from which the webhook originated, which will always be an
	// Artifactory "local repository." In cases where a Warehouse subscribes to
	// such a repository indirectly via a "virtual repository," there will be a
	// discrepancy between the inferred (local) repository URL and the URL
	// actually used by the subscription, which can prevent the receiver from
	// identifying such a Warehouse as one in need of refreshing. When specified,
	// the value of the VirtualRepoName field supersedes the value of the webhook
	// payload's `data.repo_key` field to compensate for that discrepancy.
	//
	// In practice, when using virtual repositories, a separate Artifactory
	// webhook receiver should be configured for each, but one such receiver can
	// handle inbound webhooks from any number of local repositories that are
	// aggregated by that virtual repository. For example, if a virtual repository
	// `proj-virtual` aggregates container images from all of the `proj`
	// Artifactory project's local image repositories, with a single webhook
	// configured to post to a single receiver configured for the `proj-virtual`
	// virtual repository, an image pushed to
	// `example.frog.io/proj-<local-repo-name>/<path>/image`, will cause that
	// receiver to refresh all Warehouses subscribed to
	// `example.frog.io/proj-virtual/<path>/image`.
	//
	// +optional
	VirtualRepoName string `json:"virtualRepoName,omitempty" protobuf:"bytes,2,opt,name=virtualRepoName"`
}

// AzureWebhookReceiverConfig describes a webhook receiver that is compatible
// with Azure Container Registry (ACR) and Azure DevOps payloads.
type AzureWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "system resources" namespace.
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

// GenericWebhookReceiverConfig describes a generic webhook receiver that can be
// configured to respond to any arbitrary POST by applying user-defined actions
// on user-defined sets of resources selected by name, labels and/or values in pre-built indices.
// Both types of selectors support using values extracted from the request by
// means of expressions. Currently, refreshing resources is the only supported
// action and Warehouse is the only supported kind. "Refreshing" means
// immediately enqueuing the target resource for reconciliation by its
// controller. The practical effect of refreshing a Warehouses is triggering its
// artifact discovery process.
type GenericWebhookReceiverConfig struct {
	// SecretRef contains a reference to a Secret. For Project-scoped webhook
	// receivers, the referenced Secret must be in the same namespace as the
	// ProjectConfig.
	//
	// For cluster-scoped webhook receivers, the referenced Secret must be in the
	// designated "system resources" namespace.
	//
	// The Secret's data map is expected to contain a `secret` key whose value
	// does NOT need to be shared directly with the sender. It is used only by
	// Kargo to create a complex, hard-to-guess URL, which implicitly serves as a
	// shared secret.
	//
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef" protobuf:"bytes,1,opt,name=secretRef"`

	// Actions is a list of actions to be performed when a webhook event is received.
	//
	// +kubebuilder:validation:MinItems=1
	Actions []GenericWebhookAction `json:"actions,omitempty" protobuf:"bytes,2,rep,name=actions"`
}

// GenericWebhookAction describes an action to be performed on a resource
// and the conditions under which it should be performed.
type GenericWebhookAction struct {
	// ActionType indicates the type of action to be performed. `Refresh` is the
	// only currently supported action.
	//
	// +kubebuilder:validation:Enum=Refresh;
	ActionType GenericWebhookActionType `json:"action" protobuf:"bytes,1,opt,name=action"`

	// WhenExpression defines criteria that a request must meet to run this
	// action.
	//
	// +optional
	WhenExpression string `json:"whenExpression,omitempty" protobuf:"bytes,2,opt,name=whenExpression"`

	// Parameters contains additional, action-specific parameters. Values may be
	// static or extracted from the request using expressions.
	//
	// +optional
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`

	// TargetSelectionCriteria is a list of selection criteria for the resources on which the
	// action should be performed.
	//
	// +kubebuilder:validation:MinItems=1
	TargetSelectionCriteria []GenericWebhookTargetSelectionCriteria `json:"targetSelectionCriteria,omitempty" protobuf:"bytes,4,rep,name=targets"`
}

// GenericWebhookActionType represents the type of action to be performed on a resource.
type GenericWebhookActionType string

const (
	// GenericWebhookActionTypeRefresh indicates a request to refresh the resource.
	GenericWebhookActionTypeRefresh GenericWebhookActionType = "Refresh"
)

// GenericWebhookTargetSelectionCriteria describes selection criteria for resources to which some
// action is to be applied. Name, LabelSelector, and IndexSelector are all optional
// however, at least one must be specified. When multiple criteria are specified, the
// results are the combined (logical AND) of the criteria.
type GenericWebhookTargetSelectionCriteria struct {
	// Kind is the kind of the target resource.
	//
	// +kubebuilder:validation:Enum=Warehouse;
	Kind GenericWebhookTargetKind `json:"kind" protobuf:"bytes,1,opt,name=kind"`

	// Name is the name of the target resource. If LabelSelector and/or IndexSelectors
	// are also specified, the results are the combined (logical AND) of the criteria.
	//
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`

	// LabelSelector is a label selector to identify the target resources.
	// If used with IndexSelector and/or Name, the results are the combined (logical AND) of all the criteria.
	//
	// +optional
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty" protobuf:"bytes,3,opt,name=labelSelector"`

	// IndexSelector is a selector used to identify cached target resources by cache key.
	// If used with LabelSelector and/or Name, the results are the combined (logical AND) of all the criteria.
	//
	// +optional
	IndexSelector IndexSelector `json:"indexSelector,omitempty" protobuf:"bytes,4,opt,name=indexSelector"`
}

// GenericWebhookTargetKind represents the kind of a target resource.
type GenericWebhookTargetKind string

const (
	GenericWebhookTargetKindWarehouse GenericWebhookTargetKind = "Warehouse"
)

// IndexSelector defines selection criteria that match resources on the basis of
// values in pre-built, well-known indices.
type IndexSelector struct {
	// MatchIndices is a list of index selector requirements.
	//
	// +kubebuilder:validation:MinItems=1
	MatchIndices []IndexSelectorRequirement `json:"matchIndices,omitempty" protobuf:"bytes,1,rep,name=matchIndices"`
}

// IndexSelectorRequirement encapsulates a requirement used to select indexes
// based on specific criteria.
type IndexSelectorRequirement struct {
	// Key is the key of the index.
	//
	// +kubebuilder:validation:Enum=subscribedURLs;receiverPaths
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`

	// Operator indicates the operation that should be used to evaluate
	// whether the selection requirement is satisfied.
	//
	// kubebuilder:validation:Enum=Equal;NotEqual;
	Operator IndexSelectorOperator `json:"operator" protobuf:"bytes,2,opt,name=operator"`

	// Value can be a static string or an expression that will be evaluated.
	//
	// kubebuilder:validation:Required
	Value string `json:"value" protobuf:"bytes,3,opt,name=value"`
}

// IndexSelectorOperator represents a set of operators that can be
// used in an index selector requirement.
type IndexSelectorOperator string

const (
	IndexSelectorOperatorEqual    IndexSelectorOperator = "Equals"
	IndexSelectorOperatorNotEqual IndexSelectorOperator = "NotEquals"
)

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
