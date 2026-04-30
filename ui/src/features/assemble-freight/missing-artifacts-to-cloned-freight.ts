import {
  Chart,
  DiscoveredArtifacts,
  Freight,
  GitCommit,
  Image
} from '@ui/gen/api/v1alpha1/generated_pb';

import { artifactInDiscoveredResults } from './artifact-in-discovered-results';

export const missingArtifactsToClonedFreight = (
  discoveredArtifacts: DiscoveredArtifacts | undefined,
  cloneFreight: Freight
) => {
  const missingArtifacts: (Image | GitCommit | Chart)[] = [];

  for (const image of cloneFreight?.images || []) {
    if (!artifactInDiscoveredResults(image, discoveredArtifacts)) {
      missingArtifacts.push(image);
    }
  }

  for (const chart of cloneFreight?.charts || []) {
    if (!artifactInDiscoveredResults(chart, discoveredArtifacts)) {
      missingArtifacts.push(chart);
    }
  }

  for (const commit of cloneFreight?.commits || []) {
    if (!artifactInDiscoveredResults(commit, discoveredArtifacts)) {
      missingArtifacts.push(commit);
    }
  }

  return missingArtifacts;
};
