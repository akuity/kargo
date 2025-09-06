import {
  Chart,
  DiscoveredArtifacts,
  Freight,
  GitCommit,
  Image
} from '@ui/gen/api/v1alpha1/generated_pb';

import { artifactInDiscoveredResults } from './artifact-in-discovered-results';

export const missingArtifactsToSimilarFreight = (
  discoveredArtifacts: DiscoveredArtifacts | undefined,
  similarFreight: Freight
) => {
  const missingArtifacts: (Image | GitCommit | Chart)[] = [];

  for (const image of similarFreight?.images || []) {
    if (!artifactInDiscoveredResults(image, discoveredArtifacts)) {
      missingArtifacts.push(image);
    }
  }

  for (const chart of similarFreight?.charts || []) {
    if (!artifactInDiscoveredResults(chart, discoveredArtifacts)) {
      missingArtifacts.push(chart);
    }
  }

  for (const commit of similarFreight?.commits || []) {
    if (!artifactInDiscoveredResults(commit, discoveredArtifacts)) {
      missingArtifacts.push(commit);
    }
  }

  return missingArtifacts;
};
