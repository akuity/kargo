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

// getSubscriptionName returns the optional name of the Warehouse subscription
// that produced a discovery result. The generic DiscoveryResult is excluded on
// purpose: its name IS the subscription name and already serves as its key.
export const getSubscriptionName = (res: DiscoveryResult) =>
  'subscriptionName' in res ? res.subscriptionName : undefined;

export const isEqualSubscriptions = (a: DiscoveryResult, b: DiscoveryResult) =>
  getSubscriptionKey(a) === getSubscriptionKey(b);
