import { Chart, GitCommit, Image } from '@ui/gen/api/v1alpha1/generated_pb';

import { DiscoveryResult } from './types';

export const getSubscriptionKey = (res: DiscoveryResult) => {
  if (res.$typeName === 'github.com.akuity.kargo.api.v1alpha1.DiscoveryResult') {
    return res.name;
  }

  if (res.$typeName === 'github.com.akuity.kargo.api.v1alpha1.ChartDiscoveryResult') {
    return `${res.repoURL}/${res.name}`;
  }

  return res.repoURL;
};

export const getSubscriptionKeyFreight = (res: Image | Chart | GitCommit) => {
  if (res.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Chart') {
    return `${res.repoURL}/${res.name}`;
  }

  return res.repoURL;
};

export const isEqualSubscriptions = (a: DiscoveryResult, b: DiscoveryResult) =>
  getSubscriptionKey(a) === getSubscriptionKey(b);
