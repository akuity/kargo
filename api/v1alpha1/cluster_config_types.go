package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName={clusterconfig,clusterconfigs}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// ClusterConfig is a resource type that describes cluster-level Kargo
// configuration.
type ClusterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes the configuration of a cluster.
	Spec ClusterConfigSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the current status of a ClusterConfig.
	Status ClusterConfigStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

func (c *ClusterConfig) GetStatus() *ClusterConfigStatus {
	return &c.Status
}

// ClusterConfigSpec describes cluster-level Kargo configuration.
type ClusterConfigSpec struct {
	// WebhookReceivers describes cluster-scoped webhook receivers used for
	// processing events from various external platforms
	WebhookReceivers []WebhookReceiverConfig `json:"webhookReceivers,omitempty" protobuf:"bytes,1,rep,name=webhookReceivers"`
}

// ClusterConfigStatus describes the current status of a ClusterConfig.
type ClusterConfigStatus struct {
	// Conditions contains the last observations of the ClusterConfig's current
	// state.
	//
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
	// ObservedGeneration represents the .metadata.generation that this
	// ClusterConfig was reconciled against.
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`
	// WebhookReceivers describes the status of cluster-scoped webhook receivers.
	WebhookReceivers []WebhookReceiverDetails `json:"webhookReceivers,omitempty" protobuf:"bytes,2,rep,name=webhookReceivers"`
}

// GetConditions implements the conditions.Getter interface.
func (c *ClusterConfigStatus) GetConditions() []metav1.Condition {
	return c.Conditions
}

// SetConditions implements the conditions.Setter interface.
func (c *ClusterConfigStatus) SetConditions(conditions []metav1.Condition) {
	c.Conditions = conditions
}

// +kubebuilder:object:root=true

// ClusterConfigList contains a list of ClusterConfigs.
type ClusterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []ClusterConfig `json:"items" protobuf:"bytes,2,rep,name=items"`
}
