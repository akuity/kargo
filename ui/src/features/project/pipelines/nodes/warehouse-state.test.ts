import { describe, expect, test } from 'vitest';

import { WarehouseExpanded } from '@ui/extend/types';
import { V1Condition } from '@ui/gen/api/v2/models';

import {
  getWarehouseState,
  isSubscriptionFailing,
  keepFailureThroughRetry
} from './warehouse-state';

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

  test('a Warehouse with no controller has failed, whatever it last reported', () => {
    const dead = { controllerName: 'eu-west', isControllerDead: true };

    for (const w of [
      warehouse({}),
      warehouse({
        conditions: [condition('Ready', 'True'), condition('Healthy', 'True')],
        discoveredAt: '2026-11-05T10:00:00Z'
      }),
      warehouse({ conditions: [condition('Reconciling', 'True', 'ScheduledDiscovery')] }),
      warehouse({ annotations: { 'kargo.akuity.io/refresh': 'b' }, lastHandledRefresh: 'a' }),
      warehouse({ conditions: [condition('Healthy', 'False', 'DiscoveryFailed', discoveryError)] })
    ]) {
      const state = getWarehouseState(w, dead);

      expect(state.phase).toBe('failed');
      expect(state.errorTitle).toBe('No controller');
      expect(state.errorMessage).toBe("Controller 'eu-west' is dead or nonexistent");
      // no subscription is named in it, so none is marked as the failing one
      expect(state.rawErrorMessage).toBeUndefined();
    }
  });

  test('a Warehouse on the default shard with no controller names no controller', () => {
    const state = getWarehouseState(warehouse({}), { controllerName: '', isControllerDead: true });

    expect(state.errorMessage).toBe('The default controller is dead or nonexistent');
  });

  test('a live controller leaves the state to the Warehouse', () => {
    const state = getWarehouseState(
      warehouse({ conditions: [condition('Reconciling', 'True', 'ScheduledDiscovery')] }),
      { controllerName: 'default', isControllerDead: false }
    );

    expect(state.phase).toBe('discovering');
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

describe('keepFailureThroughRetry()', () => {
  // A Warehouse at `generation`, with conditions recorded for `recordedFor`.
  const at = (
    generation: number,
    conditions: V1Condition[],
    options: { recordedFor?: number; uid?: string; annotations?: Record<string, string> } = {}
  ): WarehouseExpanded => {
    const { recordedFor = generation, uid = 'uid-1', annotations } = options;
    const w = warehouse({
      conditions: conditions.map((c) => ({ ...c, observedGeneration: recordedFor })),
      annotations
    });

    return { ...w, metadata: { ...w.metadata, uid, generation } } as WarehouseExpanded;
  };

  const failure = [
    condition('Reconciling', 'True', 'ScheduledDiscovery'),
    condition('Healthy', 'False', 'DiscoveryFailed', discoveryError)
  ];

  const discovery = [
    condition('Reconciling', 'True', 'ScheduledDiscovery'),
    condition('Ready', 'False', 'DiscoveryInProgress'),
    condition('Healthy', 'Unknown', 'Pending')
  ];

  const success = [condition('Ready', 'True'), condition('Healthy', 'True')];

  const failed = at(1, failure);

  test('a retry of a failed discovery keeps the failure shown', () => {
    expect(keepFailureThroughRetry(failed, at(1, discovery))).toBe(failed);
  });

  test('what the retry comes to is shown', () => {
    const ready = at(1, success);
    const failedAgain = at(1, failure);

    expect(keepFailureThroughRetry(failed, ready)).toBe(ready);
    expect(keepFailureThroughRetry(failed, failedAgain)).toBe(failedAgain);
  });

  test('a refresh the user asked for is shown as discovering', () => {
    const refreshing = at(1, discovery, { annotations: { 'kargo.akuity.io/refresh': 'token' } });

    expect(keepFailureThroughRetry(failed, refreshing)).toBe(refreshing);
  });

  test('the discovery of a changed spec is shown, though the edit arrives first', () => {
    // the edit, still carrying the failure recorded for the spec before it
    const edited = at(2, failure, { recordedFor: 1 });
    const discovering = at(2, discovery);

    expect(keepFailureThroughRetry(failed, edited)).toBe(edited);
    expect(keepFailureThroughRetry(edited, discovering)).toBe(discovering);
  });

  test('a retry after the changed spec failed as well keeps that failure shown', () => {
    const failedAgain = at(2, failure);

    expect(keepFailureThroughRetry(failedAgain, at(2, discovery))).toBe(failedAgain);
  });

  test('a Warehouse made again under the same name is shown as it is', () => {
    const remade = at(1, discovery, { uid: 'uid-2' });

    expect(keepFailureThroughRetry(failed, remade)).toBe(remade);
  });

  test('a discovery after no failure is shown as discovering', () => {
    const discovering = at(1, discovery);

    expect(keepFailureThroughRetry(undefined, discovering)).toBe(discovering);
    expect(keepFailureThroughRetry(at(1, success), discovering)).toBe(discovering);
  });
});
