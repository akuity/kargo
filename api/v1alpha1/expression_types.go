package v1alpha1

// ExpressionVariable describes a single variable that may be referenced by
// expressions in the context of a ClusterPromotionTask, PromotionTask,
// Promotion, AnalysisRun arguments, or other objects that support expressions.
//
// It is used to pass information to the expression evaluation engine, and to
// allow for dynamic evaluation of expressions based on the variable values.
type ExpressionVariable struct {
	// Name is the name of the variable.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=^[a-zA-Z_]\w*$
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Value is the value of the variable. It is allowed to utilize expressions
	// in the value.
	// See https://docs.kargo.io/user-guide/reference-docs/expressions for details.
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
}
