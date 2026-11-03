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
	// phaseLabel is the label under which the terminal phase value of a Promotion is
	// recorded.
	phaseLabel = "phase"
)

// durationBuckets are the histogram buckets for Promotion durations. Promotions
// commonly run from a few seconds to several minutes -- steps that wait on an
// external system (an Argo CD sync, for instance) can run considerably longer
// -- so the buckets span one second to one hour.
var durationBuckets = []float64{30, 60, 120, 300, 600, 1800, 3600}

// pendingDurationBuckets are the buckets for Promotion pending time, one
// second to one hour.
var pendingDurationBuckets = []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600}

// phaseUnknown is reported for a Promotion not yet reconciled.
const phaseUnknown = "Unknown"

var (
	metricsOnce = sync.Once{}
	promMetrics *PromotionMetrics
)

// PromotionMetrics describes Promotion activity observed by the controller. All
// of its collectors are labeled by the Project the Promotion belongs to.
type PromotionMetrics struct {
	// Completed counts Promotions that have reached a terminal phase. It is
	// additionally labeled by that phase, so a single series describes both how
	// many Promotions finished and how they finished.
	Completed *prometheus.CounterVec
	// Duration observes, in seconds, how long each Promotion spent running --
	// measured from the moment it transitioned to Running until it reached a
	// terminal phase. It is additionally labeled by outcome. Promotions that
	// end without ever having started are not observed.
	Duration *prometheus.HistogramVec
	// PendingDuration observes, in seconds, the time from a Promotion's
	// creation until it started running.
	PendingDuration *prometheus.HistogramVec
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
		Completed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metrics.Namespace,
				Name:      "promotions_completed_total",
				Help:      "Number of Promotions that have reached a terminal phase.",
			},
			[]string{metrics.ProjectLabel, phaseLabel},
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
			[]string{metrics.ProjectLabel, phaseLabel},
		),
		PendingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metrics.Namespace,
				Name:      "promotion_pending_duration_seconds",
				Help: "Time a Promotion spent waiting to run, from its creation until it " +
					"started running.",
				Buckets:                         pendingDurationBuckets,
				NativeHistogramBucketFactor:     metrics.NativeHistogramBucketFactor,
				NativeHistogramMaxBucketNumber:  metrics.NativeHistogramMaxBucketNumber,
				NativeHistogramMinResetDuration: metrics.NativeHistogramMinResetDuration,
			},
			[]string{metrics.ProjectLabel},
		),
	}

	// These are sampled at scrape time rather than recorded, so nothing needs to hold on to them
	// beyond registering them.
	nonTerminalCollector := newPromotionNonTerminalCollector(c)
	phaseCollector := newPromotionPhaseCollector(c)

	// NOTE: These must be registered with the controller-runtime registry, not the default one. The
	// manager's metrics endpoint serves only the former.
	ctrlmetrics.Registry.MustRegister(
		m.Completed,
		m.Duration,
		m.PendingDuration,
		nonTerminalCollector,
		phaseCollector,
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

// promotionPhaseCollector counts non-terminal Promotions by Project and phase
// and reports each Project's oldest pending Promotion.
type promotionPhaseCollector struct {
	client        client.Client
	byPhaseDesc   *prometheus.Desc
	oldestPending *prometheus.Desc
}

func newPromotionPhaseCollector(c client.Client) *promotionPhaseCollector {
	return &promotionPhaseCollector{
		client: c,
		byPhaseDesc: prometheus.NewDesc(
			prometheus.BuildFQName(metrics.Namespace, "", "promotions_by_phase"),
			"Number of non-terminal Promotions in each phase (Pending, Running or Unknown). "+
				"A Promotion not yet reconciled is reported in the Unknown phase.",
			[]string{metrics.ProjectLabel, phaseLabel},
			nil,
		),
		oldestPending: prometheus.NewDesc(
			prometheus.BuildFQName(metrics.Namespace, "", "promotion_oldest_pending_timestamp_seconds"),
			"Creation time, as a Unix timestamp, of the oldest Promotion that has not "+
				"started running (Pending or Unknown phase).",
			[]string{metrics.ProjectLabel},
			nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (p *promotionPhaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- p.byPhaseDesc
	ch <- p.oldestPending
}

// Collect implements prometheus.Collector.
func (p *promotionPhaseCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	promotions := &kargoapi.PromotionList{}
	// Read-only, so skip the deep copy.
	if err := p.client.List(
		ctx,
		promotions,
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionsByNonTerminalField,
				strconv.FormatBool(true),
			),
		},
		client.UnsafeDisableDeepCopy,
	); err != nil {
		logging.LoggerFromContext(ctx).Error(
			err, "Error when attempting to list promotion cache during metrics scrape",
		)
		return
	}
	summary := summarizePromotions(promotions.Items)
	for project, counts := range summary.byPhase {
		for phase, count := range counts {
			ch <- prometheus.MustNewConstMetric(
				p.byPhaseDesc, prometheus.GaugeValue, count, project, phase,
			)
		}
	}
	for project, created := range summary.oldestPending {
		ch <- prometheus.MustNewConstMetric(
			p.oldestPending,
			prometheus.GaugeValue,
			float64(created.Unix()),
			project,
		)
	}
}

// promotionSummary is what promotionPhaseCollector reports.
type promotionSummary struct {
	// byPhase counts Promotions by Project, then by phase.
	byPhase map[string]map[string]float64
	// oldestPending is the creation time of each Project's oldest Promotion
	// not yet running.
	oldestPending map[string]time.Time
}

// summarizePromotions summarizes promotions for promotionPhaseCollector.
func summarizePromotions(promotions []kargoapi.Promotion) promotionSummary {
	summary := promotionSummary{
		byPhase:       make(map[string]map[string]float64),
		oldestPending: make(map[string]time.Time),
	}
	for i := range promotions {
		promo := &promotions[i]
		phase := string(promo.Status.Phase)
		if phase == "" {
			phase = phaseUnknown
		}
		if summary.byPhase[promo.Namespace] == nil {
			summary.byPhase[promo.Namespace] = make(map[string]float64)
		}
		summary.byPhase[promo.Namespace][phase]++

		if promo.Status.Phase != "" && promo.Status.Phase != kargoapi.PromotionPhasePending {
			continue
		}
		created := promo.CreationTimestamp.Time
		if oldest, ok := summary.oldestPending[promo.Namespace]; !ok || created.Before(oldest) {
			summary.oldestPending[promo.Namespace] = created
		}
	}
	return summary
}
