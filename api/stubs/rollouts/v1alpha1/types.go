package v1alpha1

type FieldRef struct {
	// Required: Path of the field to select in the specified API version
	FieldPath string `json:"fieldPath" protobuf:"bytes,1,opt,name=fieldPath"`
}
