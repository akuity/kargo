//go:build testdata_structs
// +build testdata_structs

package testdata

type Message struct {
	WithJSONAndProtoTag          string `json:"withJSONAndProtoTag" protobuf:"bytes,1,opt,name=withJSONAndProtoTag"`
	WithUnorderedJSONAndProtoTag string `protobuf:"bytes,2,opt,name=withUnorderedJSONAndProtoTag" json:"withUnorderedJSONAndProtoTag"`
	WithJSONTag                  string `json:"withJSONTag"`
	WithIgnorableJSONTag         string `json:"-"`
	WithoutJSONTag               string
}
