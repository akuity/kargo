import { describe, expect, test } from 'vitest';

import {
  aheadFreight,
  behindCount,
  byWorstState,
  freightTally,
  inFlight,
  scopeToStage,
  stateCounts,
  targetRows
} from './target-rows';
import { entry, fleetStage, health, target } from './test-fixtures';

describe('targetRows()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }], 'f-new');
  const all = fleetStage('all-stage', [{}], 'f-new');

  test('one row per Target with its governing Stages, both sorted by name', () => {
    const rows = targetRows(
      [target('us-west-2', { region: 'us' }), target('eu-central-1', { region: 'eu' })],
      [
        entry('eu-central-1', 'all-stage', 'UpToDate'),
        entry('us-west-2', 'us-stage', 'UpToDate'),
        entry('us-west-2', 'all-stage', 'UpToDate')
      ],
      [us, all]
    );
    expect(rows.map((r) => r.target.metadata?.name)).toEqual(['eu-central-1', 'us-west-2']);
    expect(rows[0].stages.map((s) => s.stage.metadata?.name)).toEqual(['all-stage']);
    expect(rows[1].stages.map((s) => s.stage.metadata?.name)).toEqual(['all-stage', 'us-stage']);
  });

  test("reads each Stage's state from the server, and the worst across them", () => {
    const [row] = targetRows(
      [target('t', { region: 'us' })],
      [entry('t', 'all-stage', 'UpToDate'), entry('t', 'us-stage', 'NotAttempted')],
      [us, all]
    );
    expect(row.stages.map((s) => s.state)).toEqual(['UpToDate', 'NotAttempted']);
    expect(row.state).toBe('NotAttempted');
  });

  test('an ungoverned Target has never been promoted', () => {
    const [row] = targetRows([target('orphan')], [], [us]);
    expect(row.stages).toEqual([]);
    expect(row.state).toBe('NeverPromoted');
    expect(row.behind).toBe(false);
    expect(row.freight).toEqual([]);
  });

  test('behind is any governing Stage having left the Target behind, whatever its state', () => {
    const [row] = targetRows(
      [target('t')],
      [
        entry('t', 'all-stage', 'UpToDate'),
        entry('t', 'us-stage', 'Failed', { behind: true, freightName: 'f-old' })
      ],
      [us, all]
    );
    expect(row.behind).toBe(true);
    expect(row.stages.map((s) => s.behind)).toEqual([false, true]);
  });

  test('collects the Freight the Target is on across its Stages', () => {
    const [row] = targetRows(
      [target('t')],
      [
        entry('t', 'all-stage', 'UpToDate', { freightName: 'f-new' }),
        entry('t', 'us-stage', 'Behind', { freightName: 'f-old' })
      ],
      [us, all]
    );
    expect(row.freight).toEqual(['f-new', 'f-old']);
    expect(row.stages[1].currentFreight?.items?.['Warehouse/w']?.name).toBe('f-old');
  });

  test('fills in what the Freight carries when it can be resolved', () => {
    const images = [{ repoURL: 'ghcr.io/example/web', tag: '1.0.0' }];
    const [row] = targetRows(
      [target('t')],
      [entry('t', 'us-stage', 'UpToDate', { freightName: 'f-new' })],
      [us],
      (name) => ({ name, images })
    );
    expect(row.stages[0].currentFreight?.items?.['Warehouse/w']?.images).toEqual(images);
  });

  test('carries the Target health recorded for each Stage', () => {
    const [row] = targetRows(
      [target('t', {}, { status: { stages: { 'us-stage': health('Unhealthy') } } })],
      [entry('t', 'all-stage', 'UpToDate'), entry('t', 'us-stage', 'UpToDate')],
      [us, all]
    );
    expect(row.stages.map((s) => s.health?.status)).toEqual([undefined, 'Unhealthy']);
    // Health is shown beside the state, never folded into it.
    expect(row.state).toBe('UpToDate');
  });

  test('a Target the list has not caught up with is drawn from its entries', () => {
    const [row] = targetRows(
      [],
      [entry('new', 'us-stage', 'Pending', { labels: { region: 'us' } })],
      [us]
    );
    expect(row.target.metadata).toEqual({ name: 'new', labels: { region: 'us' } });
    expect(row.state).toBe('Pending');
  });

  test('an entry for a Stage not loaded yet stands in a Stage of that name', () => {
    const [row] = targetRows([target('t')], [entry('t', 'new-stage', 'Pending')], []);
    expect(row.stages[0].stage.metadata?.name).toBe('new-stage');
  });

  test('a later rollout naming only some Targets leaves the rest as they were', () => {
    // The server reads every Promotion and rollout on record, so a retry of
    // the failed Targets says nothing of the others: they keep the state the
    // rollout before it left them in.
    const rows = targetRows(
      [target('retried'), target('left-out'), target('done')],
      [
        entry('done', 'us-stage', 'UpToDate', { freightName: 'f-new' }),
        entry('left-out', 'us-stage', 'NotAttempted', {
          behind: true,
          rollout: 3,
          freightName: 'f-old'
        }),
        entry('retried', 'us-stage', 'UpToDate', { freightName: 'f-new' })
      ],
      [us]
    );
    expect(rows.map((r) => [r.target.metadata?.name, r.state])).toEqual([
      ['done', 'UpToDate'],
      ['left-out', 'NotAttempted'],
      ['retried', 'UpToDate']
    ]);
  });
});

describe('scopeToStage()', () => {
  const dev = fleetStage('dev', [{}], 'f-new');
  const prod = fleetStage('prod', [{}], 'f-new');
  const [row] = targetRows(
    [target('t')],
    [
      entry('t', 'dev', 'UpToDate', { freightName: 'f-new' }),
      entry('t', 'prod', 'Behind', { freightName: 'f-old' })
    ],
    [dev, prod]
  );

  test('reads the row as one Stage sees it, keeping every Stage that governs it', () => {
    const scoped = scopeToStage(row, row.stages[0]);
    expect(scoped.state).toBe('UpToDate');
    expect(scoped.behind).toBe(false);
    expect(scoped.freight).toEqual(['f-new']);
    expect(scoped.scopedTo).toBe(row.stages[0]);
    expect(scoped.stages).toHaveLength(2);
  });

  test("aheadFreight names what a behind Stage has moved on to, for the row's own Stage", () => {
    expect(aheadFreight(row)).toEqual(['f-new']);
    expect(aheadFreight(scopeToStage(row, row.stages[0]))).toEqual([]);
    expect(aheadFreight(scopeToStage(row, row.stages[1]))).toEqual(['f-new']);
  });
});

describe('tallies', () => {
  const s = fleetStage('s', [{}], 'f-new');
  const rows = targetRows(
    [target('a'), target('b'), target('c'), target('d')],
    [
      entry('a', 's', 'UpToDate', { freightName: 'f-new' }),
      entry('b', 's', 'Failed', { behind: true, freightName: 'f-old' }),
      entry('c', 's', 'Promoting', { freightName: 'f-old', behind: true }),
      entry('d', 's', 'UpToDate', { freightName: 'f-new' })
    ],
    [s]
  );

  test('stateCounts has every state, counted once per row', () => {
    expect(stateCounts(rows)).toEqual({
      UpToDate: 2,
      Behind: 0,
      Promoting: 1,
      Pending: 0,
      Failed: 1,
      NotAttempted: 0,
      NeverPromoted: 0
    });
  });

  test('behindCount cuts across the states', () => {
    expect(behindCount(rows)).toBe(2);
  });

  test('freightTally counts Targets per Freight, commonest first', () => {
    expect(freightTally(rows)).toEqual([
      { name: 'f-new', count: 2 },
      { name: 'f-old', count: 2 }
    ]);
  });

  test('byWorstState puts what is moving, then what went wrong, first', () => {
    expect(byWorstState(rows).map((r) => r.target.metadata?.name)).toEqual(['c', 'b', 'a', 'd']);
  });

  test('inFlight is any entry with a promotion on its way', () => {
    expect(inFlight([entry('a', 's', 'UpToDate')])).toBe(false);
    expect(inFlight([entry('a', 's', 'UpToDate'), entry('b', 's', 'Pending')])).toBe(true);
  });
});
