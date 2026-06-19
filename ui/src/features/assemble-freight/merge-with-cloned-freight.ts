import { DiscoveredArtifacts, Freight } from '@ui/gen/api/v2/models';

import {
  chartInDiscoveredResults,
  commitInDiscoveredResults,
  findChartReference,
  findCommitReference,
  findImageReference,
  imageInDiscoveredResults
} from './artifact-in-discovered-results';
import { ChosenItems } from './types';
import { getArtifactSubscriptionKey, getChartSubscriptionKey } from './unique-subscription-key';

export const mergeWithClonedFreight = (
  itemsToBeMerged: ChosenItems,
  discoveredArtifacts: DiscoveredArtifacts | undefined,
  cloneFreight: Freight
) => {
  for (const image of cloneFreight?.images || []) {
    const imageArtifact = imageInDiscoveredResults(image, discoveredArtifacts);
    if (imageArtifact) {
      itemsToBeMerged.image[getArtifactSubscriptionKey(imageArtifact)] = {
        artifact: imageArtifact,
        // it is there because then only 'imageInDiscoveredResults' omits truthy value
        info: findImageReference(image, imageArtifact.references || [])!
      };
    }
  }

  for (const chart of cloneFreight?.charts || []) {
    const chartArtifact = chartInDiscoveredResults(chart, discoveredArtifacts);

    if (chartArtifact) {
      itemsToBeMerged.chart[getChartSubscriptionKey(chartArtifact)] = {
        artifact: chartArtifact,
        // it is there because then only 'chartInDiscoveredResults' omits truthy value
        info: findChartReference(chart, chartArtifact.versions || [])!
      };
    }
  }

  for (const commit of cloneFreight?.commits || []) {
    const commitArtifact = commitInDiscoveredResults(commit, discoveredArtifacts);

    if (commitArtifact) {
      itemsToBeMerged.git[getArtifactSubscriptionKey(commitArtifact)] = {
        artifact: commitArtifact,
        // it is there because then only 'commitInDiscoveredResults' omits truthy value
        info: findCommitReference(commit, commitArtifact.commits || [])!
      };
    }
  }
};
