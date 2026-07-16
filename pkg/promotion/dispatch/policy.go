package dispatch

import (
	"embed"
	"fmt"
)

//go:embed policy/*.rego
var policyFS embed.FS

// defaultModuleName is the embedded module holding the default
// kargo.dispatch policy. A project's custom policy replaces this module and
// nothing else; the kargo.lib.* blocks always remain importable.
const defaultModuleName = "dispatch.rego"

// customModuleName names the module compiled from ProjectConfig
// spec.policy.custom, for legibility of compile errors.
const customModuleName = "projectconfig/spec/policy/custom.rego"

// policyModules returns the Rego modules to compile: the embedded standard
// library plus either the embedded default kargo.dispatch module or, when
// non-empty, the project's custom replacement.
func policyModules(custom string) (map[string]string, error) {
	entries, err := policyFS.ReadDir("policy")
	if err != nil {
		return nil, fmt.Errorf("error reading embedded policy modules: %w", err)
	}
	mods := make(map[string]string, len(entries)+1)
	for _, entry := range entries {
		if custom != "" && entry.Name() == defaultModuleName {
			continue
		}
		src, err := policyFS.ReadFile("policy/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("error reading embedded policy module %q: %w", entry.Name(), err)
		}
		mods[entry.Name()] = string(src)
	}
	if custom != "" {
		mods[customModuleName] = custom
	}
	return mods, nil
}
