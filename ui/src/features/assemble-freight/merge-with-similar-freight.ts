import {
  DiscoveredArtifacts,
  DiscoveredCommit,
  DiscoveredImageReference,
  Freight
} from '@ui/gen/api/v1alpha1/generated_pb';

import {
  artifactInDiscoveredResults,
  findChartReference,
  findCommitReference,
  findImageReference
} from './artifact-in-discovered-results';
import { DiscoveryResult, FreightInfo } from './types';
import { getSubscriptionKey } from './unique-subscription-key';

export const mergeWithSimilarFreight = (
  itemsToBeMerged: Record<string, { artifact: DiscoveryResult; info: FreightInfo }>,
  discoveredArtifacts: DiscoveredArtifacts | undefined,
  similarFreight: Freight
) => {
  for (const image of similarFreight?.images || []) {
    const imageArtifact = artifactInDiscoveredResults(image, discoveredArtifacts);
    if (
      imageArtifact &&
      imageArtifact.$typeName === 'github.com.akuity.kargo.api.v1alpha1.ImageDiscoveryResult'
    ) {
      itemsToBeMerged[getSubscriptionKey(imageArtifact)] = {
        artifact: imageArtifact,
        info: findImageReference(image, imageArtifact.references) as DiscoveredImageReference // it is there because then only 'artifactInDiscoveredResults' omits truthy value
      };
    }
  }

  for (const chart of similarFreight?.charts || []) {
    const chartArtifact = artifactInDiscoveredResults(chart, discoveredArtifacts);

    if (
      chartArtifact &&
      chartArtifact.$typeName === 'github.com.akuity.kargo.api.v1alpha1.ChartDiscoveryResult'
    ) {
      itemsToBeMerged[getSubscriptionKey(chartArtifact)] = {
        artifact: chartArtifact,
        info: findChartReference(chart, chartArtifact.versions) as string // it is there because then only 'artifactInDiscoveredResults' omits truthy value
      };
    }
  }

  for (const commit of similarFreight?.commits || []) {
    const commitArtifact = artifactInDiscoveredResults(commit, discoveredArtifacts);

    if (
      commitArtifact &&
      commitArtifact.$typeName === 'github.com.akuity.kargo.api.v1alpha1.GitDiscoveryResult'
    ) {
      itemsToBeMerged[getSubscriptionKey(commitArtifact)] = {
        artifact: commitArtifact,
        info: findCommitReference(commit, commitArtifact.commits) as DiscoveredCommit // it is there because then only 'artifactInDiscoveredResults' omits truthy value
      };
    }
  }
};
