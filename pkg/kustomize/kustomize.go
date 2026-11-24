// Package kustomize builds Kustomizations and merges YAML using Kustomize's
// libraries. Both read an OpenAPI schema that Kustomize keeps in package-level
// state and does not fully protect from concurrent use, so this package
// serializes access to it. Other code should not use those libraries directly.
//
// xref: https://github.com/kubernetes-sigs/kustomize/issues/3659
package kustomize

import (
	"fmt"
	"sync"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
)

// schemaMu serializes all use of Kustomize's package-level OpenAPI schema. A
// build resets and re-parses the schema, and a merge may re-parse it, so every
// use takes the same exclusive lock.
var schemaMu sync.Mutex

// Options configures a Build.
type Options = krusty.Options

// ResMap is the result of a Build.
type ResMap = resmap.ResMap

// Resource is a single resource in a ResMap.
type Resource = resource.Resource

// Build builds the Kustomization at path in fSys and calls use with the
// result. Every build starts from Kustomize's default OpenAPI schema, so a
// custom schema loaded by one build never affects another. A panic during the
// build or in use is returned as an error.
//
// The ResMap's methods read the schema, so use runs while Build still has
// exclusive access to it. use must not retain the ResMap, and must not call
// Build or Merge, which would wait forever for access Build is holding.
func Build(
	fSys filesys.FileSystem,
	path string,
	opts *Options,
	use func(ResMap) error,
) (err error) {
	schemaMu.Lock()
	defer schemaMu.Unlock()

	openapi.ResetOpenAPI()
	defer openapi.ResetOpenAPI()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from kustomize build panic: %v", r)
		}
	}()

	rm, err := krusty.MakeKustomizer(opts).Run(fSys, path)
	if err != nil {
		return err
	}
	return use(rm)
}

// Merge merges src into dest as kyaml's merge2.Merge does and returns the
// result. A merge waits for any build or other merge in progress.
func Merge(
	src *yaml.RNode,
	dest *yaml.RNode,
	opts yaml.MergeOptions,
) (*yaml.RNode, error) {
	schemaMu.Lock()
	defer schemaMu.Unlock()
	return merge2.Merge(src, dest, opts)
}
