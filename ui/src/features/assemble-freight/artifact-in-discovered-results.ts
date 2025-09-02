import {
  Chart,
  DiscoveredArtifacts,
  DiscoveredCommit,
  DiscoveredImageReference,
  GitCommit,
  Image
} from '@ui/gen/api/v1alpha1/generated_pb';

import { getSubscriptionKey, getSubscriptionKeyFreight } from './unique-subscription-key';

export const findImageReference = (artifact: Image, refs: DiscoveredImageReference[]) => {
  return refs?.find((imgRef) => imgRef?.tag === artifact?.tag);
};

export const findChartReference = (artifact: Chart, refs: string[]) => {
  return refs?.find((v) => v === artifact?.version);
};

export const findCommitReference = (artifact: GitCommit, refs: DiscoveredCommit[]) => {
  return refs?.find((commit) => commit?.id === artifact?.id && commit?.tag === artifact?.tag);
};

export const artifactInDiscoveredResults = (
  artifact: Image | Chart | GitCommit,
  discoveredResults?: DiscoveredArtifacts
) => {
  if (artifact?.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Image') {
    return discoveredResults?.images?.find(
      (img) =>
        getSubscriptionKey(img) === getSubscriptionKeyFreight(artifact) &&
        findImageReference(artifact, img.references)
    );
  }

  if (artifact?.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Chart') {
    return discoveredResults?.charts?.find(
      (chart) =>
        getSubscriptionKey(chart) === getSubscriptionKeyFreight(artifact) &&
        findChartReference(artifact, chart.versions)
    );
  }

  if (artifact.$typeName === 'github.com.akuity.kargo.api.v1alpha1.GitCommit') {
    return discoveredResults?.git?.find(
      (git) =>
        getSubscriptionKey(git) === getSubscriptionKeyFreight(artifact) &&
        findCommitReference(artifact, git.commits)
    );
  }

  return false;
};
