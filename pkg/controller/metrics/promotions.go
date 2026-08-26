// Package metrics defines the Prometheus metrics exported by Kargo's
// controller. Collectors are registered with the controller-runtime registry,
// which backs the metrics endpoint served by the controller's manager.
package metrics

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/metrics"
)

const (
	// terminalPhaseLabel is the label under which the terminal phase value of a Promotion is
	// recorded.
	terminalPhaseLabel = "terminalPhase"
)

// durationBuckets are the histogram buckets for Promotion durations. Promotions
// commonly run from a few seconds to several minutes -- steps that wait on an
// external system (an Argo CD sync, for instance) can run considerably longer
// -- so the buckets span one second to one hour.
var durationBuckets = []float64{30, 60, 120, 300, 600, 1800, 3600}

var (
	metricsOnce = sync.Once{}
	promMetrics *PromotionMetrics
)

// PromotionMetrics describes Promotion activity observed by the controller. All
// of its collectors are labeled by the Project the Promotion belongs to.
type PromotionMetrics struct {
	// Created counts Promotions the controller has observed being created.
	Created *prometheus.CounterVec
	// RetryableErrors counts errors encountered while executing Promotions that
	// a subsequent attempt may recover from. A single Promotion can contribute
	// more than once.
	RetryableErrors *prometheus.CounterVec
	// TerminalErrors counts Promotions that have errored unrecoverably.
	TerminalErrors *prometheus.CounterVec
	// Duration observes, in seconds, how long each Promotion spent running --
	// measured from the moment it transitioned to Running until it reached a
	// terminal phase. It is additionally labeled by outcome. Promotions that
	// end without ever having started are not observed.
	Duration *prometheus.HistogramVec
}

// NewPromotionMetrics returns PromotionMetrics whose collectors have been registered with the
// controller-runtime registry. It panics if any of the collectors is already registered. Calling
// this function multiple times will return the same PromotionMetrics instance.
func NewPromotionMetrics(
	c client.Client,
) *PromotionMetrics {
	metricsOnce.Do(func() {
		promMetrics = initPromotionMetrics(c)
	})
	return promMetrics
}

func initPromotionMetrics(
	c client.Client,
) *PromotionMetrics {
	m := &PromotionMetrics{
		Created: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metrics.Namespace,
				Name:      "promotions_created_total",
				Help:      "Number of Promotions the controller has observed being created.",
			},
			[]string{metrics.ProjectLabel},
		),
		RetryableErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metrics.Namespace,
				Name:      "promotions_errored_retryable_total",
				Help:      "Number of retryable errors encountered while executing Promotions.",
			},
			[]string{metrics.ProjectLabel},
		),
		TerminalErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metrics.Namespace,
				Name:      "promotions_errored_terminal_total",
				Help:      "Number of Promotions that have terminally errored.",
			},
			[]string{metrics.ProjectLabel},
		),
		Duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metrics.Namespace,
				Name:      "promotion_duration_seconds",
				Help: "Time a Promotion spent running, from the moment it started " +
					"until it reached a terminal phase.",
				Buckets: durationBuckets,
				// Because of the wide range of promotion durations, we also enable Native
				// Histograms if the Prometheus server supports them. Notes on each of the settings
				// is in the main metrics package

				NativeHistogramBucketFactor:     metrics.NativeHistogramBucketFactor,
				NativeHistogramMaxBucketNumber:  metrics.NativeHistogramMaxBucketNumber,
				NativeHistogramMinResetDuration: metrics.NativeHistogramMinResetDuration,
			},
			[]string{metrics.ProjectLabel, terminalPhaseLabel},
		),
	}

	// This one is sampled at scrape time rather than recorded, so nothing needs to hold on to it
	// beyond registering it.
	nonTerminalCollector := newPromotionNonTerminalCollector(c)

	// NOTE: These must be registered with the controller-runtime registry, not the default one. The
	// manager's metrics endpoint serves only the former.
	ctrlmetrics.Registry.MustRegister(
		m.Created,
		m.RetryableErrors,
		m.TerminalErrors,
		m.Duration,
		nonTerminalCollector,
	)
	return m
}

// promotionNonTerminalCollector counts non-terminal Promotions by Project. It implements
// prometheus.Collector rather than using a GaugeFunc because a GaugeFunc's labels have to be known
// when it is constructed, and Projects are created and deleted while the controller runs.
type promotionNonTerminalCollector struct {
	client client.Client
	desc   *prometheus.Desc
}

func newPromotionNonTerminalCollector(c client.Client) *promotionNonTerminalCollector {
	return &promotionNonTerminalCollector{
		client: c,
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(metrics.Namespace, "", "promotions_non_terminal"),
			"Number of Promotions that are in a non-terminal state (Pending or Running).",
			[]string{metrics.ProjectLabel},
			nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (p *promotionNonTerminalCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- p.desc
}

// Collect implements prometheus.Collector. Unlike the Stage collector, this one can still lean on
// an index: non-terminal Promotions are typically a small fraction of all Promotions, so selecting
// them up front keeps a scrape from walking every Promotion the cache holds.
func (p *promotionNonTerminalCollector) Collect(ch chan<- prometheus.Metric) {
	promotions := &kargoapi.PromotionList{}
	// Create a background context with a timeout to avoid blocking on a return or in case of
	// shutdown. We're ok with not plumbing through a top-level context here because this is
	// short and really doesn't matter if it is interrupted
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	if err := p.client.List(
		ctx,
		promotions,
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionsByNonTerminalField,
				strconv.FormatBool(true),
			),
		},
	); err != nil {
		logging.LoggerFromContext(ctx).Error(
			err, "Error when attempting to list promotion cache during metrics scrape",
		)
		return
	}

	for project, count := range countPromotionsByProject(promotions.Items) {
		ch <- prometheus.MustNewConstMetric(p.desc, prometheus.GaugeValue, count, project)
	}
}

// countPromotionsByProject buckets the given Promotions by the Project they belong to. Only the
// Projects actually represented are reported, so a Project's series disappears once it has no
// Promotions left in the list.
func countPromotionsByProject(promotions []kargoapi.Promotion) map[string]float64 {
	counts := make(map[string]float64)
	for _, promotion := range promotions {
		counts[promotion.Namespace]++
	}
	return counts
}
