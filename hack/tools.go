//go:build tools
// +build tools

// This file is for managing Go programs version with `go.mod`

package tools

import (
	_ "k8s.io/code-generator/cmd/go-to-protobuf"
	_ "k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo"
)
