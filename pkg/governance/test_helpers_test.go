package governance

import (
	"github.com/goccy/go-yaml"
)

// mustAction constructs an action from a kind name and an arbitrary
// config value (the same value that would appear under the kind key in
// YAML). It panics if the value can't be marshaled. Intended for use in
// test fixtures, where panics are acceptable and the alternative —
// hand-constructing yaml byte slices — is noisy.
func mustAction(kind string, configValue any) action {
	cfg, err := yaml.Marshal(configValue)
	if err != nil {
		panic(err)
	}
	return action{kind: kind, config: cfg}
}
