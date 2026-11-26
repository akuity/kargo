import { describe, expect, test } from 'vitest';

import type { Project, V1Condition } from '@ui/gen/api/v2/models';

// Relative: vitest resolves @ui paths to files but not to a directory's index.
import { WarehouseColorMapHex } from '../../../stage/utils';

import { healthSummary, projectAccent, projectReadiness } from './project-card-utils';

const condition = (c: Partial<V1Condition>): V1Condition => ({
  lastTransitionTime: '',
  message: '',
  reason: '',
  status: 'True',
  type: 'Ready',
  ...c
});

const projectWith = (conditions?: V1Condition[]): Project => ({ status: { conditions } });

describe('projectAccent', () => {
  test('picks a color from the Warehouse palette', () => {
    expect(Object.values(WarehouseColorMapHex)).toContain(projectAccent('kargo-demo'));
  });

  test('gives the same name the same color every time', () => {
    expect(projectAccent('kargo-demo')).toBe(projectAccent('kargo-demo'));
  });

  test('spreads names across the palette', () => {
    const names = ['alpha', 'bravo', 'charlie', 'delta', 'echo', 'foxtrot', 'golf', 'hotel'];
    expect(new Set(names.map(projectAccent)).size).toBeGreaterThan(1);
  });

  test('copes with an empty name', () => {
    expect(Object.values(WarehouseColorMapHex)).toContain(projectAccent(''));
  });
});

describe('projectReadiness', () => {
  test('a project without conditions is ready', () => {
    expect(projectReadiness(projectWith())).toEqual({ state: 'ready' });
    expect(projectReadiness(undefined)).toEqual({ state: 'ready' });
  });

  test('a Ready project is ready', () => {
    expect(projectReadiness(projectWith([condition({})]))).toEqual({ state: 'ready' });
  });

  test('a project still being set up is syncing', () => {
    const project = projectWith([
      condition({ status: 'False', reason: 'Syncing', message: 'Ensuring project namespace' })
    ]);
    expect(projectReadiness(project)).toEqual({
      state: 'syncing',
      message: 'Ensuring project namespace'
    });
  });

  test('an Unknown Ready condition is syncing', () => {
    expect(projectReadiness(projectWith([condition({ status: 'Unknown' })])).state).toBe('syncing');
  });

  test('a project that could not be set up has failed', () => {
    const project = projectWith([
      condition({ status: 'False', reason: 'EnsuringNamespaceFailed', message: 'boom' })
    ]);
    expect(projectReadiness(project)).toEqual({ state: 'failed', message: 'boom' });
  });

  test('a Stalled project has failed, whatever Ready says', () => {
    const project = projectWith([
      condition({ status: 'False', reason: 'Syncing' }),
      condition({ type: 'Stalled', reason: 'ExistingNamespaceMissingLabel', message: 'label' })
    ]);
    expect(projectReadiness(project)).toEqual({ state: 'failed', message: 'label' });
  });
});

describe('healthSummary', () => {
  test('no stats is an empty summary', () => {
    expect(healthSummary(undefined)).toEqual({
      count: 0,
      healthy: 0,
      allHealthy: false
    });
  });

  test('some healthy', () => {
    expect(healthSummary({ count: 4, health: { healthy: 3 } })).toEqual({
      count: 4,
      healthy: 3,
      allHealthy: false
    });
  });

  test('all healthy', () => {
    expect(healthSummary({ count: 2, health: { healthy: 2 } }).allHealthy).toBe(true);
  });

  test('a healthy count missing from the stats is zero', () => {
    expect(healthSummary({ count: 3 })).toMatchObject({ healthy: 0 });
  });

  test('never reports more healthy than there are', () => {
    expect(healthSummary({ count: 2, health: { healthy: 5 } })).toMatchObject({
      healthy: 2,
      allHealthy: true
    });
  });
});
