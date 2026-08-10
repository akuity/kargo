import { Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { DiscoveryResult } from './types';

export const getSubscriptionKey = (res: DiscoveryResult) => {
  if ('artifactReferences' in res) {
    return res.name || '';
  }

  if ('name' in res && 'repoURL' in res) {
    return `${res.repoURL}/${res.name}`;
  }

  if ('repoURL' in res) {
    return res.repoURL || '';
  }

  return '';
};

export const getSubscriptionKeyFreight = (res: Image | Chart | GitCommit) => {
  return getSubscriptionKey(res);
};

export const isEqualSubscriptions = (a: DiscoveryResult, b: DiscoveryResult) =>
  getSubscriptionKey(a) === getSubscriptionKey(b);
