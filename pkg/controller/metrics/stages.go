// Package metrics defines the Prometheus metrics exported by Kargo's
// controller. Collectors are registered with the controller-runtime registry,
// which backs the metrics endpoint served by the controller's manager.
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/metrics"
)

const (
	// reasonLabel is the label under which the reason on a Stage's Ready condition is recorded.
	reasonLabel = "reason"
)

// reasonUnknown is the reason reported for a Stage that has no Ready condition yet, because the
// controller has not reconciled it since it was created.
const reasonUnknown = "Unknown"

var stageMetricsOnce = sync.Once{}

// RegisterStageMetrics registers the Stage metrics with the controller-runtime registry. They are
// all sampled from the cache at scrape time rather than recorded to, so there is nothing for a
// caller to hold on to and nothing is returned. It panics if any of the collectors is already
// registered. Calling this function more than once registers nothing further.
func RegisterStageMetrics(c client.Client) {
	stageMetricsOnce.Do(func() {
		// NOTE: These must be registered with the controller-runtime registry, not the default one.
		// The manager's metrics endpoint serves only the former.
		ctrlmetrics.Registry.MustRegister(newStageReadyCollector(c))
	})
}

// stageReadyCollector counts Stages by Project and by the reason on their Ready condition. It
// implements prometheus.Collector rather than using a GaugeFunc because a GaugeFunc's labels have
// to be known when it is constructed, and both Projects and reasons vary at runtime.
type stageReadyCollector struct {
	client client.Client
	desc   *prometheus.Desc
}

func newStageReadyCollector(c client.Client) *stageReadyCollector {
	return &stageReadyCollector{
		client: c,
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(metrics.Namespace, "", "stages_by_ready_reason"),
			"Number of Stages in each readiness state, by the reason on their Ready condition. A "+
				"Stage is counted exactly once. Control flow Stages are omitted: their Ready "+
				"condition is set directly and never accounts for health or verification, so the "+
				"reason they report conveys no useful information.",
			[]string{metrics.ProjectLabel, reasonLabel},
			nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (s *stageReadyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.desc
}

// Collect implements prometheus.Collector. It lists every Stage once and buckets them in memory.
// There is no index to lean on here the way the Promotion gauge does: breaking counts down by
// Project means every Stage has to be seen anyway.
func (s *stageReadyCollector) Collect(ch chan<- prometheus.Metric) {
	// Create a background context with a timeout to avoid blocking on a return or in case of
	// shutdown. We're ok with not plumbing through a top-level context here because this is
	// short and really doesn't matter if it is interrupted
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	stages := &kargoapi.StageList{}
	if err := s.client.List(ctx, stages); err != nil {
		logging.LoggerFromContext(ctx).Error(
			err, "Error when attempting to list stage cache during metrics scrape",
		)
		return
	}

	for project, counts := range countStagesByReadyReason(stages.Items) {
		for reason, count := range counts {
			ch <- prometheus.MustNewConstMetric(
				s.desc,
				prometheus.GaugeValue,
				count,
				project,
				reason,
			)
		}
	}
}

// countStagesByReadyReason buckets Stages by Project and by the reason on their Ready condition.
// The reconciler collapses everything it knows about a Stage into that one condition in a fixed
// precedence, so the reason already names the most important thing true of the Stage. See
// summarizeConditions in pkg/controller/stages.
//
// Control flow Stages are skipped. Only the states actually observed are reported, so a series
// disappears once no Stage is in that state.
func countStagesByReadyReason(
	stages []kargoapi.Stage,
) map[string]map[string]float64 {
	counts := make(map[string]map[string]float64)
	for i := range stages {
		stage := &stages[i]
		if stage.IsControlFlow() {
			continue
		}

		reason := reasonUnknown
		if ready := conditions.Get(&stage.Status, kargoapi.ConditionTypeReady); ready != nil {
			reason = ready.Reason
		}

		if counts[stage.Namespace] == nil {
			counts[stage.Namespace] = make(map[string]float64)
		}
		counts[stage.Namespace][reason]++
	}
	return counts
}
