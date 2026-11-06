import { AnalysisStatus } from '@ui/features/common/analysis-modal/types';
import { RolloutsAnalysisRun } from '@ui/gen/api/v2/models';

export const extractFilters = (ar?: RolloutsAnalysisRun) => {
  const metrics = ar?.spec?.metrics?.filter((metric) => !!metric?.provider?.job);

  const containerNames: Record<string, string[]> = {};

  for (const metric of metrics || []) {
    const metricName = metric?.name;

    if (!metricName) {
      continue;
    }

    const containers = metric?.provider?.job?.spec?.template?.spec?.containers;

    for (const container of containers || []) {
      if (!containerNames[metricName]) {
        containerNames[metricName] = [];
      }

      if (container?.name) {
        containerNames[metricName].push(container.name);
      }
    }
  }

  return {
    jobNames: metrics?.map((metric) => metric?.name).filter((name): name is string => !!name) || [],
    containerNames
  };
};

// Once an AnalysisRun is in one of these phases, no more Jobs will start.
const completedPhases: string[] = [
  AnalysisStatus.Successful,
  AnalysisStatus.Failed,
  AnalysisStatus.Error,
  AnalysisStatus.Inconclusive
];

// isJobPending returns whether the Job for the named metric of the given
// AnalysisRun is yet to start. Until it does, the metric has no measurement
// recording the Job's name, so there are no logs to request.
export const isJobPending = (ar?: RolloutsAnalysisRun, metricName?: string) => {
  if (!ar || !metricName || completedPhases.includes(ar.status?.phase || '')) {
    return false;
  }

  const result = ar.status?.metricResults?.find((r) => r?.name === metricName);

  return !result?.measurements?.some((m) => !!m?.metadata?.['job-name']);
};

// hasPendingJob returns whether the Job for any job metric of the given
// AnalysisRun is yet to start.
export const hasPendingJob = (ar?: RolloutsAnalysisRun) =>
  extractFilters(ar).jobNames.some((name) => isJobPending(ar, name));

export const pendingJobRefetchIntervalMs = 5000;

// refetchWhileJobPending is a refetchInterval for an AnalysisRun query. It
// keeps the AnalysisRun fresh until the Jobs for all of its metrics have
// started, so their logs can be requested as soon as there are any. It stops
// if the query fails, so that a failing read does not raise an error
// notification on every tick.
export const refetchWhileJobPending = (query: {
  state: { status: string; data?: { data?: RolloutsAnalysisRun } };
}) =>
  query.state.status !== 'error' && hasPendingJob(query.state.data?.data)
    ? pendingJobRefetchIntervalMs
    : false;
