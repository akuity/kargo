import { describe, expect, test } from 'vitest';

import { WarehouseExpanded } from '@ui/extend/types';
import { V1Condition } from '@ui/gen/api/v2/models';

import { getWarehouseState, isSubscriptionFailing } from './warehouse-state';

const condition = (type: string, status: string, reason = '', message = ''): V1Condition => ({
  type,
  status,
  reason,
  message,
  lastTransitionTime: '2026-11-05T10:00:00Z'
});

const warehouse = (partial: {
  conditions?: V1Condition[];
  annotations?: Record<string, string>;
  lastHandledRefresh?: string;
  discoveredAt?: string;
}): WarehouseExpanded =>
  ({
    metadata: { namespace: 'my-project', name: 'nginx', annotations: partial.annotations },
    status: {
      conditions: partial.conditions,
      lastHandledRefresh: partial.lastHandledRefresh,
      discoveredArtifacts: partial.discoveredAt ? { discoveredAt: partial.discoveredAt } : undefined
    }
  }) as WarehouseExpanded;

const discoveryError =
  'Unable to discover artifacts: error discovering artifacts: error discovering newest ' +
  'applicable images "ghcr.io/nginxinc/nginx-unprivileged": unauthorized';

describe('getWarehouseState()', () => {
  test('a healthy Warehouse is ready, with when it last discovered', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [condition('Ready', 'True'), condition('Healthy', 'True')],
        discoveredAt: '2026-11-05T10:00:00Z'
      })
    );

    expect(state.phase).toBe('ready');
    expect(state.discoveredAt).toEqual(new Date('2026-11-05T10:00:00Z'));
  });

  test('a Warehouse with no status is ready, with no discovery to report', () => {
    const state = getWarehouseState(warehouse({}));

    expect(state.phase).toBe('ready');
    expect(state.discoveredAt).toBeUndefined();
  });

  test('a reconciling Warehouse is discovering', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [
          condition('Reconciling', 'True', 'ScheduledDiscovery'),
          condition('Ready', 'False', 'DiscoveryInProgress'),
          condition('Healthy', 'Unknown', 'Pending')
        ]
      })
    );

    expect(state.phase).toBe('discovering');
  });

  test('an unhandled refresh is discovering', () => {
    const state = getWarehouseState(
      warehouse({
        annotations: { 'kargo.akuity.io/refresh': 'b' },
        lastHandledRefresh: 'a'
      })
    );

    expect(state.phase).toBe('discovering');
  });

  test('a handled refresh is not', () => {
    const state = getWarehouseState(
      warehouse({
        annotations: { 'kargo.akuity.io/refresh': 'a' },
        lastHandledRefresh: 'a'
      })
    );

    expect(state.phase).toBe('ready');
  });

  test('a retry wins over a failure still on record', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [condition('Healthy', 'False', 'DiscoveryFailed', discoveryError)],
        annotations: { 'kargo.akuity.io/refresh': 'b' },
        lastHandledRefresh: 'a'
      })
    );

    expect(state.phase).toBe('discovering');
  });

  // What the controller leaves behind when discovery fails: the Reconciling
  // condition it set going in is never cleared.
  test('a failed discovery still reporting reconciling has failed', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [
          condition('Reconciling', 'True', 'ScheduledDiscovery'),
          condition('Ready', 'False', 'DiscoveryFailure', 'Artifact discovery failed: ...'),
          condition('Healthy', 'False', 'DiscoveryFailed', discoveryError)
        ],
        annotations: { 'kargo.akuity.io/refresh': 'a' },
        lastHandledRefresh: 'a'
      })
    );

    expect(state.phase).toBe('failed');
    expect(state.errorTitle).toBe('Discovery failed');
  });

  test('a failed Freight build still reporting reconciling has failed', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [
          condition('Reconciling', 'True', 'FreightCreationInProgress'),
          condition('Ready', 'False', 'FreightBuildFailure', 'boom'),
          condition('Healthy', 'False', 'FreightBuildFailure', 'boom')
        ]
      })
    );

    expect(state.phase).toBe('failed');
    expect(state.errorTitle).toBe('Freight creation failed');
  });

  test('an unhealthy Warehouse failed, with the redundant prefixes trimmed', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [
          condition('Ready', 'False', 'DiscoveryFailure', `Artifact discovery failed: ...`),
          condition('Healthy', 'False', 'DiscoveryFailed', discoveryError)
        ]
      })
    );

    expect(state.phase).toBe('failed');
    expect(state.errorTitle).toBe('Discovery failed');
    expect(state.errorMessage).toBe(
      'error discovering newest applicable images "ghcr.io/nginxinc/nginx-unprivileged": unauthorized'
    );
    expect(state.rawErrorMessage).toBe(discoveryError);
  });

  test('a failure to make Freight is not called a discovery failure', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [condition('Healthy', 'False', 'FreightBuildFailure', 'boom')]
      })
    );

    expect(state.phase).toBe('failed');
    expect(state.errorTitle).toBe('Freight creation failed');
    expect(state.errorMessage).toBe('boom');
  });

  test('not ready with no reconciling condition is a failure', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [condition('Ready', 'False', 'MissingArtifacts', 'No artifacts discovered')]
      })
    );

    expect(state.phase).toBe('failed');
    expect(state.errorMessage).toBe('No artifacts discovered');
  });

  test('not ready while reconciling reports is not', () => {
    const state = getWarehouseState(
      warehouse({
        conditions: [
          condition('Reconciling', 'False', 'Done'),
          condition('Ready', 'False', 'AwaitingFreightCreation')
        ]
      })
    );

    expect(state.phase).toBe('ready');
  });
});

describe('isSubscriptionFailing()', () => {
  const failed = getWarehouseState(
    warehouse({ conditions: [condition('Healthy', 'False', 'DiscoveryFailed', discoveryError)] })
  );

  test('the subscription whose repository the error quotes is failing', () => {
    expect(
      isSubscriptionFailing(failed, { image: { repoURL: 'ghcr.io/nginxinc/nginx-unprivileged' } })
    ).toBe(true);
  });

  test('a subscription whose repository only prefixes the quoted one is not', () => {
    expect(isSubscriptionFailing(failed, { image: { repoURL: 'ghcr.io/nginxinc/nginx' } })).toBe(
      false
    );
  });

  test('a subscription with no repository is not', () => {
    expect(isSubscriptionFailing(failed, { subscription: { subscriptionType: 'custom' } })).toBe(
      false
    );
  });

  test('nothing is failing on a Warehouse that has not failed', () => {
    expect(
      isSubscriptionFailing(getWarehouseState(warehouse({})), {
        image: { repoURL: 'ghcr.io/nginxinc/nginx-unprivileged' }
      })
    ).toBe(false);
  });
});
