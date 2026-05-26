import { DiscoveredCommit, DiscoveredImageReference } from '@ui/gen/api/v1alpha1/generated_pb';
import { DiscoveredArtifacts, Freight } from '@ui/gen/api/v2/models';

import {
  artifactInDiscoveredResults,
  findChartReference,
  findCommitReference,
  findImageReference
} from './artifact-in-discovered-results';
import { DiscoveryResult, FreightInfo } from './types';
import { getSubscriptionKey } from './unique-subscription-key';

export const mergeWithClonedFreight = (
  itemsToBeMerged: Record<string, { artifact: DiscoveryResult; info: FreightInfo }>,
  discoveredArtifacts: DiscoveredArtifacts | undefined,
  cloneFreight: Freight
) => {
  for (const image of cloneFreight?.images || []) {
    const imageArtifact = artifactInDiscoveredResults(image, discoveredArtifacts);
    if (imageArtifact && 'references' in imageArtifact) {
      itemsToBeMerged[getSubscriptionKey(imageArtifact)] = {
        artifact: imageArtifact,
        info: findImageReference(image, imageArtifact.references || []) as DiscoveredImageReference // it is there because then only 'artifactInDiscoveredResults' omits truthy value
      };
    }
  }

  for (const chart of cloneFreight?.charts || []) {
    const chartArtifact = artifactInDiscoveredResults(chart, discoveredArtifacts);

    if (chartArtifact && 'versions' in chartArtifact) {
      itemsToBeMerged[getSubscriptionKey(chartArtifact)] = {
        artifact: chartArtifact,
        info: findChartReference(chart, chartArtifact.versions || []) as string // it is there because then only 'artifactInDiscoveredResults' omits truthy value
      };
    }
  }

  for (const commit of cloneFreight?.commits || []) {
    const commitArtifact = artifactInDiscoveredResults(commit, discoveredArtifacts);

    if (commitArtifact && 'commits' in commitArtifact) {
      itemsToBeMerged[getSubscriptionKey(commitArtifact)] = {
        artifact: commitArtifact,
        info: findCommitReference(commit, commitArtifact.commits || []) as DiscoveredCommit // it is there because then only 'artifactInDiscoveredResults' omits truthy value
      };
    }
  }
};
