import { Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { isArtifactChart } from './artifact-type-guards';
import { DiscoveryResult } from './types';

export const getSubscriptionKey = (res: DiscoveryResult) => {
  if ('artifactReferences' in res) {
    return res.name || '';
  }

  if ('versions' in res) {
    return `${res.repoURL}/${res.name}`;
  }

  if ('repoURL' in res) {
    return res.repoURL || '';
  }

  return '';
};

export const getSubscriptionKeyFreight = (res: Image | Chart | GitCommit) => {
  if (isArtifactChart(res)) {
    return `${res.repoURL}/${res.name}`;
  }

  return res.repoURL;
};

export const isEqualSubscriptions = (a: DiscoveryResult, b: DiscoveryResult) =>
  getSubscriptionKey(a) === getSubscriptionKey(b);
