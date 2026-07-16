package dispatch

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
)

//go:embed policy
var policyFS embed.FS

// customModuleName names the module compiled from ProjectConfig
// spec.policy.custom, for legibility of compile errors.
const customModuleName = "projectconfig/spec/policy/custom.rego"

// customPackagePath is the package a custom policy module must declare. The
// default kargo.dispatch module gathers data.kargo.custom.violation, and
// kargo.lib.exclusions honors data.kargo.custom.exclusions_bypass; both are
// inert when no custom module is present.
const customPackagePath = "kargo.custom"

// policyModules returns the Rego modules to compile: the embedded standard
// library and default kargo.dispatch module, plus, when non-empty, the
// project's custom kargo.custom module, which composes into (never
// replaces) the default policy.
func policyModules(custom string) (map[string]string, error) {
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
	if custom != "" {
		if err = validateCustomModule(custom); err != nil {
			return nil, err
		}
		mods[customModuleName] = custom
	}
	return mods, nil
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

// validateCustomModule requires the custom module to declare the well-known
// kargo.custom package. Any other package could silently shadow or conflict
// with the default policy or the standard library.
func validateCustomModule(custom string) error {
	mod, err := ast.ParseModule(customModuleName, custom)
	if err != nil {
		return fmt.Errorf("error parsing custom policy module: %w", err)
	}
	if got := strings.TrimPrefix(mod.Package.Path.String(), "data."); got != customPackagePath {
		return fmt.Errorf(
			"custom policy module declares package %q; must be %q",
			got, customPackagePath,
		)
	}
	return nil
}
