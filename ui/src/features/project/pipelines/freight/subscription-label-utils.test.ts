import { describe, expect, test } from 'vitest';

import { RepoSubscription, WarehouseExpanded } from '@ui/extend/types';

import { getShowSubscriptionNames, getSubscriptionLabel } from './subscription-label-utils';

const subscriptions = [
  { name: 'frontend', image: { repoURL: 'ghcr.io/acme/guestbook' } },
  { name: 'config', git: { repoURL: 'https://github.com/acme/config.git' } },
  { name: 'api-chart', chart: { repoURL: 'https://charts.acme.io', name: 'api' } },
  { name: 'web-chart', chart: { repoURL: 'https://charts.acme.io', name: 'web' } }
] as RepoSubscription[];

describe('getSubscriptionLabel', () => {
  test('prefers the name recorded on the artifact', () => {
    expect(
      getSubscriptionLabel(
        { repoURL: 'ghcr.io/acme/guestbook', tag: 'v1', subscriptionName: 'recorded' },
        subscriptions
      )
    ).toBe('recorded');
  });

  test("falls back to the name of the artifact's subscription in the Warehouse", () => {
    expect(
      getSubscriptionLabel({ repoURL: 'ghcr.io/acme/guestbook', tag: 'v1' }, subscriptions)
    ).toBe('frontend');
    expect(
      getSubscriptionLabel(
        { repoURL: 'https://github.com/acme/config.git', id: 'abc' },
        subscriptions
      )
    ).toBe('config');
  });

  test('matches a chart from a Helm repository by its name too', () => {
    expect(
      getSubscriptionLabel(
        { repoURL: 'https://charts.acme.io', name: 'web', version: '1.0.0' },
        subscriptions
      )
    ).toBe('web-chart');
  });

  test("falls back to the repository's short name, as the subscription node does", () => {
    expect(
      getSubscriptionLabel({ repoURL: 'ghcr.io/acme/unnamed', tag: 'v1' }, subscriptions)
    ).toBe('unnamed');
    expect(
      getSubscriptionLabel({ repoURL: 'ghcr.io/acme/guestbook', tag: 'v1' }, [
        { image: { repoURL: 'ghcr.io/acme/guestbook' } } as RepoSubscription
      ])
    ).toBe('guestbook');
  });

  test('uses the artifact type for a generic artifact without a name', () => {
    expect(getSubscriptionLabel({ artifactType: 'oci', version: '1' })).toBe('oci');
  });
});

describe('getShowSubscriptionNames', () => {
  const warehouse = (subscriptionCount: number) =>
    ({
      spec: {
        subscriptions: Array.from({ length: subscriptionCount }, (_, i) => ({
          image: { repoURL: `ghcr.io/acme/image-${i}` }
        }))
      }
    }) as WarehouseExpanded;

  test('is on by default when a Warehouse has more than four subscriptions', () => {
    expect(getShowSubscriptionNames(undefined, [warehouse(2), warehouse(5)])).toBe(true);
  });

  test('is off by default when no Warehouse has more than four', () => {
    expect(getShowSubscriptionNames(undefined, [warehouse(4), warehouse(1)])).toBe(false);
    expect(getShowSubscriptionNames(undefined, [])).toBe(false);
  });

  test("follows the user's choice once they make one", () => {
    expect(getShowSubscriptionNames(false, [warehouse(5)])).toBe(false);
    expect(getShowSubscriptionNames(true, [warehouse(1)])).toBe(true);
  });
});
