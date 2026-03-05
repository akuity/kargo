//go:build tools
// +build tools

// This file is for managing Go programs version with `go.mod`, which allows
// them to be kept up-to-date through tools like Dependabot.

package tools

import (
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/go-swagger/go-swagger/cmd/swagger"
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/swaggo/swag/cmd/swag"
	_ "golang.org/x/tools/cmd/goimports"
	_ "helm.sh/helm/v3/cmd/helm"
	_ "k8s.io/code-generator/cmd/go-to-protobuf"
	_ "k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
