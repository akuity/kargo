//go:build testdata_generated
// +build testdata_generated

package testdata

// +kubebuilder:object:root=true
type Message struct {
	WithJSONAndProtoTag          string `json:"withJSONAndProtoTag,omitempty" protobuf:"bytes,1,opt,name=withJSONAndProtoTag"`
	WithUnorderedJSONAndProtoTag string `json:"withUnorderedJSONAndProtoTag,omitempty" protobuf:"bytes,2,opt,name=withUnorderedJSONAndProtoTag"`
	WithJSONTag                  string `json:"withJSONTag,omitempty" protobuf:"bytes,3,opt,name=withJSONTag"`
	WithIgnorableJSONTag         string `json:"-"`
	WithoutJSONTag               string
}
