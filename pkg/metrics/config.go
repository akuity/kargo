package metrics

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Family is a group of metrics that METRICS_DISABLE can turn off.
type Family string

const (
	// FamilyRESTClient is the metrics EnableRESTClientMetrics registers.
	FamilyRESTClient Family = "rest-client"
	// FamilyLifecycle is the controller's lifecycle metrics.
	FamilyLifecycle Family = "lifecycle"
)

// familiesAll disables every family.
const familiesAll = "all"

// families are the known families.
var families = []Family{FamilyRESTClient, FamilyLifecycle}

// Config configures which metric families are registered.
type Config struct {
	// Disable lists the families not to register, or "all".
	Disable []string `envconfig:"METRICS_DISABLE"`
	// disabled is Disable, validated and expanded.
	disabled []Family
}

// ConfigFromEnv returns a Config from the environment. An unknown family is an
// error.
func ConfigFromEnv() (Config, error) {
	cfg := Config{}
	envconfig.MustProcess("", &cfg)
	for _, name := range cfg.Disable {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		if name == familiesAll {
			cfg.disabled = append(cfg.disabled, families...)
			continue
		}
		family := Family(name)
		if !slices.Contains(families, family) {
			return Config{}, fmt.Errorf(
				"unknown metrics family %q in METRICS_DISABLE; known families: %s, or %s",
				name,
				joinFamilies(families),
				familiesAll,
			)
		}
		cfg.disabled = append(cfg.disabled, family)
	}
	return cfg, nil
}

// Enabled returns whether family is to be registered.
func (c Config) Enabled(family Family) bool {
	return !slices.Contains(c.disabled, family)
}

// joinFamilies returns fs, comma-separated.
func joinFamilies(fs []Family) string {
	names := make([]string, len(fs))
	for i, f := range fs {
		names[i] = string(f)
	}
	return strings.Join(names, ", ")
}
