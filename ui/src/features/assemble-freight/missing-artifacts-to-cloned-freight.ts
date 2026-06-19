import { Chart, DiscoveredArtifacts, Freight, GitCommit, Image } from '@ui/gen/api/v2/models';

import {
  chartInDiscoveredResults,
  commitInDiscoveredResults,
  imageInDiscoveredResults
} from './artifact-in-discovered-results';

export type MissingArtifacts = {
  images: Image[];
  charts: Chart[];
  commits: GitCommit[];
};

export const missingArtifactsToClonedFreight = (
  discoveredArtifacts: DiscoveredArtifacts | undefined,
  cloneFreight: Freight | undefined
): MissingArtifacts => {
  const missingArtifacts: MissingArtifacts = { images: [], charts: [], commits: [] };

  for (const image of cloneFreight?.images || []) {
    if (!imageInDiscoveredResults(image, discoveredArtifacts)) {
      missingArtifacts.images.push(image);
    }
  }

  for (const chart of cloneFreight?.charts || []) {
    if (!chartInDiscoveredResults(chart, discoveredArtifacts)) {
      missingArtifacts.charts.push(chart);
    }
  }

  for (const commit of cloneFreight?.commits || []) {
    if (!commitInDiscoveredResults(commit, discoveredArtifacts)) {
      missingArtifacts.commits.push(commit);
    }
  }

  return missingArtifacts;
};
