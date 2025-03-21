import { AnalysisRun } from '@ui/gen/api/stubs/rollouts/v1alpha1/generated_pb';

export const extractFilters = (ar: AnalysisRun) => {
  const metrics = ar?.spec?.metrics?.filter((metric) => !!metric?.provider?.job);

  const containerNames: Record<string, string[]> = {};

  for (const metric of metrics || []) {
    const containers = metric?.provider?.job?.spec?.template?.spec?.containers;

    for (const container of containers || []) {
      if (!containerNames[metric?.name]) {
        containerNames[metric?.name] = [];
      }

      containerNames[metric?.name].push(container?.name);
    }
  }

  return {
    jobNames: metrics?.map((metric) => metric?.name) || [],
    containerNames
  };
};
