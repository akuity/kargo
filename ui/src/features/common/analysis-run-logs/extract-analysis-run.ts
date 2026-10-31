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

// Resolved on every render rather than frozen at mount, because the
// AnalysisRun may arrive after the component does, and URL params may name a
// metric or container (even the literal "undefined") that doesn't exist.
export const resolveSelection = (
  filterableItems: ReturnType<typeof extractFilters>,
  requested: { selectedJob?: string; selectedContainer?: string }
) => {
  const selectedJob =
    requested.selectedJob && filterableItems.jobNames.includes(requested.selectedJob)
      ? requested.selectedJob
      : filterableItems.jobNames[0];

  const containers = selectedJob ? filterableItems.containerNames[selectedJob] : undefined;

  const selectedContainer =
    requested.selectedContainer && containers?.includes(requested.selectedContainer)
      ? requested.selectedContainer
      : containers?.[0];

  return { selectedJob, selectedContainer };
};
