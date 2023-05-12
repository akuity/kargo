//go:build tools

package tools

import (
	_ "github.com/gogo/protobuf/gogoproto"
	_ "k8s.io/code-generator/cmd/go-to-protobuf"
	_ "k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo"
)
