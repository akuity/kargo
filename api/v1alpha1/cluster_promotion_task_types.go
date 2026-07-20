package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:resource:scope=Cluster,shortName={clusterpromotask,clusterpromotasks}
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

type ClusterPromotionTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the desired transition of a specific Stage into a specific
	// Freight.
	//
	// +kubebuilder:validation:Required
	Spec PromotionTaskSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// ClusterPromotionTaskList contains a list of PromotionTasks.
type ClusterPromotionTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPromotionTask `json:"items"`
}
