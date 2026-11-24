// Package metrics defines the Prometheus metrics exported by Kargo's
// controller. Collectors are registered with the controller-runtime registry,
// which backs the metrics endpoint served by the controller's manager.
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/akuity/kargo/pkg/metrics"
)

const (
	// stepKindLabel is the label under which the kind of Step is recorded.
	stepKindLabel = "stepKind"
	// terminalStatusLabel is the label under which the step's terminal status is recorded:
	// Succeeded, Failed (a terminal, non-technical failure), or Errored (a technical failure,
	// which is retried until the step's error threshold is met). Skipped steps are not recorded.
	terminalStatusLabel = "terminalStatus"
)

// durationBuckets are the histogram buckets for step durations. Steps can vary widely in how long
// they take to run. Some are just a few seconds, but some that wait (like waiting for a PR) can
// take a long time. For this reason we also enable Native Histograms if the Prometheus server
// supports them, which will allow us to get a more accurate view of the distribution of step
// durations.
var durationBuckets = []float64{5, 10, 30, 60, 120, 300, 600, 1800, 3600}

var (
	metricsOnce = sync.Once{}
	stepMetrics *StepMetrics
)

// StepMetrics describes Step activity observed by the promotion engine. All
// of its collectors are labeled by the Project the Step belongs to.
type StepMetrics struct {
	// Duration observes, in seconds, how long each step took to run, labeled by the project they
	// belong to, the step kind, and the terminal status the step ended in.
	Duration *prometheus.HistogramVec
}

// NewStepMetrics returns StepMetrics whose collectors have been registered with the
// controller-runtime registry. It panics if any of the collectors is already registered. Calling
// this function multiple times will return the same StepMetrics instance.
func NewStepMetrics() *StepMetrics {
	metricsOnce.Do(func() {
		stepMetrics = initStepMetrics()
	})
	return stepMetrics
}

func initStepMetrics() *StepMetrics {
	m := &StepMetrics{
		Duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metrics.Namespace,
				Name:      "promotion_step_duration_seconds",
				Help: "Time a Promotion step spent running, from the moment it started until it reached a " +
					"terminal phase. It is recommended to scrape Native Histograms instead of the static " +
					"buckets, if the Prometheus server supports them.",
				Buckets: durationBuckets,
				// Because of the wide range of step durations, we also enable Native Histograms if
				// the Prometheus server supports them. Notes on each of the settings is in the main
				// metrics package

				NativeHistogramBucketFactor:     metrics.NativeHistogramBucketFactor,
				NativeHistogramMaxBucketNumber:  metrics.NativeHistogramMaxBucketNumber,
				NativeHistogramMinResetDuration: metrics.NativeHistogramMinResetDuration,
			},
			// NOTE(thomastaylor312): Project label is the only one of these labels that is truly
			// unbounded. Steps will continue to increase, but slowly and won't ever be massive. If
			// we start having cardinality issues, project is probably the first thing to drop
			[]string{metrics.ProjectLabel, stepKindLabel, terminalStatusLabel},
		),
	}

	// NOTE: These must be registered with the controller-runtime registry, not the default one. The
	// manager's metrics endpoint serves only the former.
	ctrlmetrics.Registry.MustRegister(
		m.Duration,
	)
	return m
}
