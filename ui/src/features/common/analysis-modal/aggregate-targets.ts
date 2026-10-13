import {
  AnalysisRunRequest,
  RolloutsAnalysisRunSpec,
  RolloutsAnalysisRunStatus,
  RolloutsMetric,
  RolloutsMetricResult,
  RolloutsRunSummary
} from '@ui/gen/api/v2/models';

// aggregateTargets flattens the per-target statuses of an AnalysisRunRequest
// into a single AnalysisRun-shaped spec/status pair so the existing
// AnalysisModal rendering (summary + metric panels) can be reused. When a
// request has more than one target, metric names are prefixed with the target
// name to keep them distinct in the aggregated view.
export const aggregateTargets = (
  request?: AnalysisRunRequest
): [RolloutsAnalysisRunSpec, RolloutsAnalysisRunStatus] => {
  const targets = request?.status?.analysisRuns ?? [];
  const prefixMetrics = targets.length > 1;

  const metricResults: RolloutsMetricResult[] = [];
  const metrics: RolloutsMetric[] = [];
  const runSummary: Required<RolloutsRunSummary> = {
    count: 0,
    error: 0,
    failed: 0,
    inconclusive: 0,
    successful: 0
  };

  for (const target of targets) {
    const summary = target.status?.runSummary;
    runSummary.count += summary?.count ?? 0;
    runSummary.error += summary?.error ?? 0;
    runSummary.failed += summary?.failed ?? 0;
    runSummary.inconclusive += summary?.inconclusive ?? 0;
    runSummary.successful += summary?.successful ?? 0;

    for (const result of target.status?.metricResults ?? []) {
      const name = prefixMetrics
        ? `${target.targetName}: ${result.name ?? ''}`
        : (result.name ?? '');
      metricResults.push({ ...result, name });
      // A minimal spec entry is enough for transformMetrics to surface the
      // metric; thresholds/queries are only derived when a provider is present,
      // which these synthesized specs intentionally omit.
      metrics.push({ name });
    }
  }

  const spec: RolloutsAnalysisRunSpec = { metrics, args: [] };
  const status: RolloutsAnalysisRunStatus = {
    phase: request?.status?.phase,
    message: request?.status?.message,
    startedAt: request?.status?.startedAt,
    completedAt: request?.status?.completedAt,
    metricResults,
    runSummary
  };

  return [spec, status];
};
