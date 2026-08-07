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
