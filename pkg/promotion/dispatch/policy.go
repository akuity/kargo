package dispatch

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
)

//go:embed policy
var policyFS embed.FS

// Module names for the custom policy sources, for legibility of compile
// errors. Note that reported line numbers are offset by the prepended
// header (see customHeader).
const (
	projectCustomModuleName = "projectconfig/spec/customPolicy.rego"
	clusterCustomModuleName = "clusterconfig/spec/customPolicy.rego"
)

// customHeader is prepended to every custom policy source: users write
// rules only. The named package's shipped module (policy/kargo/<pkg>)
// supplies inert defaults for the hook points, and the aliased import
// puts the building blocks (kargo.is_forward, kargo.is_semver_patch; see
// policy/kargo/lib/lib.rego) one qualifier away (an unused import is
// fine; the engine compiles non-strict).
const customHeader = `package kargo.%s

import rego.v1

import data.kargo.lib as kargo

`

// packageDeclPattern spots a package declaration at the start of a line,
// which a rules-only custom source must not contain.
var packageDeclPattern = regexp.MustCompile(`(?m)^\s*package\s`)

// policyModules returns the Rego modules to compile: the embedded standard
// library and default kargo.dispatch module, plus the project's and the
// cluster's custom sources (either may be empty), each prepended with the
// standard header so they compose into -- never replace -- the default
// policy.
func policyModules(projectCustom, clusterCustom string) (map[string]string, error) {
	mods := map[string]string{}
	err := fs.WalkDir(
		policyFS,
		"policy",
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".rego") {
				return nil
			}
			src, err := policyFS.ReadFile(path)
			if err != nil {
				return err
			}
			mods[strings.TrimPrefix(path, "policy/")] = string(src)
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error reading embedded policy modules: %w", err)
	}
	if projectCustom != "" {
		src, err := buildCustomModule("project", projectCustom)
		if err != nil {
			return nil, fmt.Errorf("invalid ProjectConfig customPolicy: %w", err)
		}
		mods[projectCustomModuleName] = src
	}
	if clusterCustom != "" {
		src, err := buildCustomModule("cluster", clusterCustom)
		if err != nil {
			return nil, fmt.Errorf("invalid ClusterConfig customPolicy: %w", err)
		}
		mods[clusterCustomModuleName] = src
	}
	return mods, nil
}

// buildCustomModule prepends the standard header (package kargo.<pkg> and
// the library imports) to a rules-only custom source.
func buildCustomModule(pkg, custom string) (string, error) {
	if packageDeclPattern.MatchString(custom) {
		return "", fmt.Errorf(
			"customPolicy contains only rules; the package declaration and " +
				"standard imports are provided automatically",
		)
	}
	return fmt.Sprintf(customHeader, pkg) + custom, nil
}

// policySchemas returns the embedded JSON Schemas describing the policy
// input and data documents. Each schemas/<name>.json is registered as
// schema.<name>, for use in module metadata annotations; annotated modules
// (the standard library) are type-checked against them at compile time.
func policySchemas() (*ast.SchemaSet, error) {
	entries, err := policyFS.ReadDir("policy/schemas")
	if err != nil {
		return nil, fmt.Errorf("error reading embedded policy schemas: %w", err)
	}
	ss := ast.NewSchemaSet()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		raw, err := policyFS.ReadFile("policy/schemas/" + name)
		if err != nil {
			return nil, fmt.Errorf("error reading embedded policy schema %q: %w", name, err)
		}
		var doc any
		if err = json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("error parsing embedded policy schema %q: %w", name, err)
		}
		ref := ast.MustParseRef("schema." + strings.TrimSuffix(name, path.Ext(name)))
		ss.Put(ref, doc)
	}
	return ss, nil
}
