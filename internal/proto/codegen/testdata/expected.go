//go:build testdata_expected
// +build testdata_expected

package testdata

// +kubebuilder:object:root=true
type Message struct {
	WithJSONAndProtoTag          string `json:"withJSONAndProtoTag" protobuf:"bytes,1,opt,name=withJSONAndProtoTag"`
	WithJSONOmitEmptyAndProtoTag string `json:"withJSONOmitEmptyAndProtoTag,omitempty" protobuf:"bytes,2,opt,name=withUnorderedJSONAndProtoTag"`
	WithJSONTag                  string `json:"withJSONTag" protobuf:"bytes,3,opt,name=withJSONTag"`
	WithJSONOmitEmptyTag         string `json:"withJSONOmitEmptyTag,omitempty" protobuf:"bytes,4,opt,name=withJSONOmitEmptyTag"`
	WithIgnorableJSONTag         string `json:"-"`
	WithoutJSONTag               string
}
