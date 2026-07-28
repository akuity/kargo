import {
  Chart,
  DiscoveredArtifacts,
  DiscoveredCommit,
  DiscoveredImageReference,
  GitCommit,
  Image
} from '@ui/gen/api/v2/models';

import { isArtifactChart, isArtifactGitCommit, isArtifactImage } from './artifact-type-guards';
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
  if (isArtifactImage(artifact)) {
    return discoveredResults?.images?.find(
      (img) =>
        getSubscriptionKey(img) === getSubscriptionKeyFreight(artifact) &&
        findImageReference(artifact, img.references ?? [])
    );
  }

  if (isArtifactChart(artifact)) {
    return discoveredResults?.charts?.find(
      (chart) =>
        getSubscriptionKey(chart) === getSubscriptionKeyFreight(artifact) &&
        findChartReference(artifact, chart.versions ?? [])
    );
  }

  if (isArtifactGitCommit(artifact)) {
    return discoveredResults?.git?.find(
      (git) =>
        getSubscriptionKey(git) === getSubscriptionKeyFreight(artifact) &&
        findCommitReference(artifact, git.commits ?? [])
    );
  }

  return false;
};
