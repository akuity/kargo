//go:build testdata_structs
// +build testdata_structs

package testdata

// +kubebuilder:object:root=true
type Message struct {
	WithJSONAndProtoTag          string `json:"withJSONAndProtoTag" protobuf:"bytes,1,opt,name=withJSONAndProtoTag"`
	WithJSONOmitEmptyAndProtoTag string `json:"withJSONOmitEmptyAndProtoTag,omitempty" protobuf:"bytes,2,opt,name=withUnorderedJSONAndProtoTag"`
	WithJSONTag                  string `json:"withJSONTag"`
	WithJSONOmitEmptyTag         string `json:"withJSONOmitEmptyTag,omitempty"`
	WithIgnorableJSONTag         string `json:"-"`
	WithoutJSONTag               string
}
