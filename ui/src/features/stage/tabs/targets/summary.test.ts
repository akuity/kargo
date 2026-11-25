import { describe, expect, test } from 'vitest';

import { TargetStatusEntry } from '../../../../gen/api/v2/models';
import { targetRows } from '../../../project/targets/model/target-rows';
import { entry, fleetStage, health, target } from '../../../project/targets/model/test-fixtures';

import {
  assessedHealth,
  governing,
  healthTally,
  neverPromotedNote,
  owningTargetSet
} from './summary';

const stage = fleetStage('prod', [{ matchLabels: { env: 'prod' } }], 'freight-new');

// rows builds the tab's rows the way the tab does: from this one Stage's
// statuses, with the Targets filling in their own health.
const rows = (targets: { name: string; health?: string }[], entries?: TargetStatusEntry[]) =>
  targetRows(
    targets.map(({ name, health: status }) =>
      target(name, { env: 'prod' }, status ? { status: { stages: { prod: health(status) } } } : {})
    ),
    entries ?? targets.map(({ name }) => entry(name, 'prod', 'UpToDate')),
    [stage]
  );

describe('governing()', () => {
  test('is where a Target stands with the one Stage of the tab', () => {
    const [row] = rows([{ name: 't', health: 'Healthy' }]);
    expect(governing(row)?.stage.metadata?.name).toBe('prod');
    expect(governing(row)?.state).toBe('UpToDate');
    expect(governing(row)?.health?.status).toBe('Healthy');
  });

  test('is nothing for a Target the Stage has no status for', () => {
    const [row] = rows([{ name: 't' }], []);
    expect(governing(row)).toBeUndefined();
  });
});

describe('healthTally()', () => {
  test('counts what has been assessed apart from what has not', () => {
    const tally = healthTally(
      rows([
        { name: 'a', health: 'Healthy' },
        { name: 'b', health: 'Unhealthy' },
        { name: 'c' },
        { name: 'd' }
      ])
    );
    // The breakdown counts the assessed Targets and no others, so it sums to
    // the figure the tile prints above it.
    expect(tally.assessed).toBe(2);
    expect(tally.byStatus).toEqual([
      { status: 'Healthy', count: 1 },
      { status: 'Unhealthy', count: 1 }
    ]);
    expect(tally.unassessed).toBe(2);
  });

  test('an unassessed fleet has nothing to break down', () => {
    const tally = healthTally(rows([{ name: 'a' }, { name: 'b' }]));
    expect(tally).toEqual({ assessed: 0, byStatus: [], unassessed: 2 });
  });
});

describe('assessedHealth()', () => {
  test('is false until something has assessed a Target', () => {
    expect(assessedHealth(rows([{ name: 'a' }, { name: 'b' }]))).toBe(false);
  });

  test('is true as soon as one Target carries a status', () => {
    expect(assessedHealth(rows([{ name: 'a' }, { name: 'b', health: 'Unknown' }]))).toBe(true);
  });
});

describe('owningTargetSet()', () => {
  test('names the TargetSet a Target was generated from', () => {
    expect(
      owningTargetSet({
        metadata: {
          name: 't',
          ownerReferences: [
            { apiVersion: 'v1', kind: 'ConfigMap', name: 'not-it', uid: '1' },
            {
              apiVersion: 'ee.kargo.akuity.io/v1alpha1',
              kind: 'TargetSet',
              name: 'clusters',
              uid: '2'
            }
          ]
        }
      })
    ).toBe('clusters');
  });

  test('is nothing for a Target written by hand', () => {
    expect(owningTargetSet(target('t'))).toBeUndefined();
  });
});

describe('neverPromotedNote()', () => {
  test('says every Target has Freight only when every row has some', () => {
    const all = rows(
      [{ name: 'a' }, { name: 'b' }],
      [
        entry('a', 'prod', 'UpToDate', { freightName: 'freight-new' }),
        entry('b', 'prod', 'Behind', { freightName: 'freight-old' })
      ]
    );
    expect(neverPromotedNote(0, all)).toBe('all have Freight');
  });

  test('says no Freight yet when the never promoted are the only Targets without', () => {
    const some = rows(
      [{ name: 'a' }, { name: 'b' }],
      [
        entry('a', 'prod', 'UpToDate', { freightName: 'freight-new' }),
        entry('b', 'prod', 'NeverPromoted')
      ]
    );
    expect(neverPromotedNote(1, some)).toBe('no Freight yet');
  });

  test('counts Targets without Freight that are not never promoted', () => {
    // A failed first promotion and a Target a rollout has yet to reach leave
    // no Freight behind, though neither counts as never promoted.
    const none = rows(
      [{ name: 'a' }, { name: 'b' }, { name: 'c' }],
      [
        entry('a', 'prod', 'UpToDate', { freightName: 'freight-new' }),
        entry('b', 'prod', 'Failed'),
        entry('c', 'prod', 'Pending')
      ]
    );
    expect(neverPromotedNote(0, none)).toBe('2 have no Freight');
    expect(neverPromotedNote(0, none.slice(0, 2))).toBe('1 has no Freight');
  });
});
