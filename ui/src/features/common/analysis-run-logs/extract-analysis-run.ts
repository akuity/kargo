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

const isComplete = (ar: RolloutsAnalysisRun) => completedPhases.includes(ar.status?.phase || '');

// isJobRecorded returns whether the Job for the named metric of the given
// AnalysisRun has been recorded where the API server looks for it. The server
// finds a Job's logs using the namespace and name recorded on the metric's
// first measurement, so until that measurement records both, there are no logs
// to request.
const isJobRecorded = (ar: RolloutsAnalysisRun, metricName: string) => {
  const result = ar.status?.metricResults?.find((r) => r?.name === metricName);
  const metadata = result?.measurements?.[0]?.metadata;

  return !!metadata?.['job-namespace'] && !!metadata?.['job-name'];
};

// isJobPending returns whether the Job for the named metric of the given
// AnalysisRun is yet to start.
export const isJobPending = (ar?: RolloutsAnalysisRun, metricName?: string) =>
  !!ar && !!metricName && !isComplete(ar) && !isJobRecorded(ar, metricName);

// isJobMissing returns whether the given AnalysisRun completed without the Job
// for the named metric ever starting, as when verification is aborted while the
// Job is still delayed. Such a metric will never have logs.
export const isJobMissing = (ar?: RolloutsAnalysisRun, metricName?: string) =>
  !!ar && !!metricName && isComplete(ar) && !isJobRecorded(ar, metricName);

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
