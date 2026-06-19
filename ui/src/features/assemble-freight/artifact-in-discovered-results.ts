import {
  Chart,
  DiscoveredArtifacts,
  DiscoveredCommit,
  DiscoveredImageReference,
  GitCommit,
  Image
} from '@ui/gen/api/v2/models';

import { getArtifactSubscriptionKey, getChartSubscriptionKey } from './unique-subscription-key';

export const findImageReference = (artifact: Image, refs: DiscoveredImageReference[]) => {
  return refs?.find((imgRef) => imgRef?.tag === artifact?.tag);
};

export const findChartReference = (artifact: Chart, refs: string[]) => {
  return refs?.find((v) => v === artifact?.version);
};

export const findCommitReference = (artifact: GitCommit, refs: DiscoveredCommit[]) => {
  return refs?.find((commit) => commit?.id === artifact?.id && commit?.tag === artifact?.tag);
};

export const imageInDiscoveredResults = (
  artifact: Image,
  discoveredResults?: DiscoveredArtifacts
) =>
  discoveredResults?.images?.find(
    (img) =>
      getArtifactSubscriptionKey(img) === getArtifactSubscriptionKey(artifact) &&
      findImageReference(artifact, img.references ?? [])
  );

export const chartInDiscoveredResults = (
  artifact: Chart,
  discoveredResults?: DiscoveredArtifacts
) =>
  discoveredResults?.charts?.find(
    (chart) =>
      getChartSubscriptionKey(chart) === getChartSubscriptionKey(artifact) &&
      findChartReference(artifact, chart.versions ?? [])
  );

export const commitInDiscoveredResults = (
  artifact: GitCommit,
  discoveredResults?: DiscoveredArtifacts
) =>
  discoveredResults?.git?.find(
    (git) =>
      getArtifactSubscriptionKey(git) === getArtifactSubscriptionKey(artifact) &&
      findCommitReference(artifact, git.commits ?? [])
  );
