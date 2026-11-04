package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// metricsFamilyRESTClient names, in METRICS_DISABLE, the client-go REST
// client metrics enableRESTClientMetrics registers.
const metricsFamilyRESTClient = "rest-client"

// metricsFamiliesAll, in METRICS_DISABLE, names every family in
// metricsFamilies.
const metricsFamiliesAll = "all"

// metricsFamilies are the metric families METRICS_DISABLE can name.
var metricsFamilies = []string{metricsFamilyRESTClient}

// metricsConfig is the configuration of the metrics every component
// registers.
type metricsConfig struct {
	// Disable lists, comma-separated, the metric families Kargo registers that
	// are not to be registered, or "all" for every one of them. All are
	// registered by default. controller-runtime's own metrics are not affected.
	Disable []string `envconfig:"METRICS_DISABLE"`
}

// metricsConfigFromEnv returns a metricsConfig populated from the
// environment, with "all" expanded to every family. Families are matched
// case-insensitively; an unknown one is an error, so that a typo doesn't leave
// a family on.
func metricsConfigFromEnv() (metricsConfig, error) {
	cfg := metricsConfig{}
	envconfig.MustProcess("", &cfg)
	disable := make([]string, 0, len(cfg.Disable))
	for _, family := range cfg.Disable {
		family = strings.ToLower(strings.TrimSpace(family))
		if family == "" {
			continue
		}
		if family == metricsFamiliesAll {
			disable = append(disable, metricsFamilies...)
			continue
		}
		if !slices.Contains(metricsFamilies, family) {
			return metricsConfig{}, fmt.Errorf(
				"unknown metrics family %q in METRICS_DISABLE; known families: %s, or %s",
				family,
				strings.Join(metricsFamilies, ", "),
				metricsFamiliesAll,
			)
		}
		disable = append(disable, family)
	}
	cfg.Disable = disable
	return cfg, nil
}

// restClientMetricsEnabled returns whether the client-go REST client metrics
// are to be registered.
func (c metricsConfig) restClientMetricsEnabled() bool {
	return !slices.Contains(c.Disable, metricsFamilyRESTClient)
}
