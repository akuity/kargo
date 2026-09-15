import { describe, expect, test } from 'vitest';

import { PromotionRequest, Stage, Target } from '@ui/gen/api/v2/models';

import {
  SEVERITY_ACTIVE,
  SEVERITY_FAILED,
  SEVERITY_SETTLED,
  fleetRows,
  freightNames,
  matchesTarget,
  rowPhase,
  rowSeverity,
  rowsSummary
} from './fleet-utils';

const stage = (partial: Partial<Stage> = {}): Stage => ({
  metadata: { name: 'prod' },
  ...partial
});

const target = (name: string, partial: Partial<Target> = {}): Target => ({
  metadata: { name },
  ...partial
});

const request = (targets: string[], status?: PromotionRequest['status']): PromotionRequest => ({
  metadata: { name: 'prod.01a.x' },
  spec: { stage: 'prod', freight: 'f1', targets: targets.map((t) => ({ name: t })) },
  status
});

describe('rowSeverity()', () => {
  test.each([
    ['Failed', SEVERITY_FAILED],
    ['Errored', SEVERITY_FAILED],
    ['Pending', SEVERITY_ACTIVE],
    ['Running', SEVERITY_ACTIVE],
    ['Succeeded', SEVERITY_SETTLED],
    ['Aborted', SEVERITY_SETTLED],
    [undefined, SEVERITY_SETTLED]
  ])('%s -> %s', (phase, severity) => {
    expect(rowSeverity({ target: target('t'), included: true, phase })).toBe(severity);
  });
});

describe('fleetRows()', () => {
  test('no Targets yields no rows', () => {
    expect(fleetRows(stage(), [], request(['t']))).toEqual([]);
  });

  test('a Target the Stage has never promoted to', () => {
    const [row] = fleetRows(stage(), [target('t')]);
    expect(row).toMatchObject({ included: false });
    expect(row.request).toBeUndefined();
    expect(row.phase).toBeUndefined();
    expect(row.currentFreight).toBeUndefined();
    expect(row.health).toBeUndefined();
  });

  test('a Target that joined after the round was created', () => {
    const round = request(['other']);
    const [row] = fleetRows(stage(), [target('t')], round);
    expect(row).toMatchObject({ request: round, included: false });
    expect(row.phase).toBeUndefined();
  });

  test('reads the child Promotion from the round', () => {
    const round = request(['t'], {
      phase: 'Running',
      targets: [{ name: 't', promotion: 'prod.t.01a.x', phase: 'Succeeded' }]
    });
    expect(fleetRows(stage(), [target('t')], round)[0]).toMatchObject({
      included: true,
      promotion: 'prod.t.01a.x',
      phase: 'Succeeded'
    });
  });

  test('a child that exists without a phase yet is Pending', () => {
    const round = request(['t'], {
      phase: 'Running',
      targets: [{ name: 't', promotion: 'prod.t.01a.x' }]
    });
    expect(fleetRows(stage(), [target('t')], round)[0]).toMatchObject({
      promotion: 'prod.t.01a.x',
      phase: 'Pending'
    });
  });

  test('is Pending while a live round has not reached the Target', () => {
    expect(fleetRows(stage(), [target('t')], request(['t'], { phase: 'Running' }))[0].phase).toBe(
      'Pending'
    );
  });

  test('takes the round phase when the round ended before creating a child', () => {
    expect(fleetRows(stage(), [target('t')], request(['t'], { phase: 'Errored' }))[0].phase).toBe(
      'Errored'
    );
  });

  test('reads current Freight and health from the Target status for this Stage', () => {
    const t = target('t', {
      status: {
        stages: {
          prod: {
            currentFreight: { id: 'c1', items: { 'Warehouse/w': { name: 'f1' } } },
            health: { status: 'Healthy' }
          },
          other: {
            currentFreight: { id: 'c9', items: { 'Warehouse/w': { name: 'f9' } } },
            health: { status: 'Unhealthy' }
          }
        }
      }
    });
    const [row] = fleetRows(stage(), [t]);
    expect(row.currentFreight?.id).toBe('c1');
    expect(row.health?.status).toBe('Healthy');
  });

  test('upToDate compares the Target collection to the Stage current collection', () => {
    const s = stage({ status: { freightHistory: [{ id: 'c1', items: {} }] } });
    const on = target('on', {
      status: { stages: { prod: { currentFreight: { id: 'c1', items: {} } } } }
    });
    const behind = target('behind', {
      status: { stages: { prod: { currentFreight: { id: 'c0', items: {} } } } }
    });
    const unknown = target('unknown');
    const rows = fleetRows(s, [on, behind, unknown]);
    const byName = Object.fromEntries(rows.map((row) => [row.target.metadata?.name, row]));
    expect(byName.on.upToDate).toBe(true);
    expect(byName.behind.upToDate).toBe(false);
    expect(byName.unknown.upToDate).toBeUndefined();
  });

  test('upToDate is undefined when the Stage has no current collection', () => {
    const on = target('on', {
      status: { stages: { prod: { currentFreight: { id: 'c1', items: {} } } } }
    });
    expect(fleetRows(stage(), [on])[0].upToDate).toBeUndefined();
  });

  test('sorts trouble first, then by name', () => {
    const round = request(['a', 'b', 'c', 'd'], {
      phase: 'Running',
      targets: [
        { name: 'a', promotion: 'p', phase: 'Succeeded' },
        { name: 'b', promotion: 'p', phase: 'Errored' },
        { name: 'c', promotion: 'p', phase: 'Running' }
      ]
    });
    const rows = fleetRows(stage(), [target('d'), target('c'), target('b'), target('a')], round);
    expect(rows.map((row) => row.target.metadata?.name)).toEqual(['b', 'c', 'd', 'a']);
  });
});

describe('freightNames()', () => {
  test('lists each origin Freight by name, sorted', () => {
    expect(
      freightNames({
        id: 'c',
        items: { 'Warehouse/b': { name: 'zz' }, 'Warehouse/a': { name: 'aa' } }
      })
    ).toEqual(['aa', 'zz']);
  });

  test('handles an absent collection', () => {
    expect(freightNames(undefined)).toEqual([]);
  });
});

describe('rowsSummary()', () => {
  test('counts each phase and skips rows the round did not name', () => {
    const row = (phase?: string, included = true) => ({
      target: target('t'),
      included,
      phase
    });
    expect(
      rowsSummary([
        row('Succeeded'),
        row('Succeeded'),
        row('Errored'),
        row('Running'),
        row('Pending'),
        row('Failed'),
        row('Aborted'),
        row(undefined, false)
      ])
    ).toEqual({ succeeded: 2, errored: 1, running: 1, pending: 1, failed: 1, aborted: 1 });
  });

  test('no rows yields an empty summary', () => {
    expect(rowsSummary([])).toEqual({});
  });
});

const rowOf = (name: string, phase?: string, labels?: Record<string, string>) => ({
  target: target(name, { metadata: { name, labels } }),
  included: true,
  phase
});

describe('matchesTarget()', () => {
  const row = rowOf('eu-north-1', 'Errored', { region: 'eu', flaky: 'true' });

  test('an empty needle matches', () => {
    expect(matchesTarget(row, '')).toBe(true);
    expect(matchesTarget(row, '   ')).toBe(true);
  });

  test('matches name and label key=value, case-insensitively', () => {
    expect(matchesTarget(row, 'EU-north')).toBe(true);
    expect(matchesTarget(row, 'flaky=true')).toBe(true);
    expect(matchesTarget(row, 'region=')).toBe(true);
    expect(matchesTarget(row, 'us-east')).toBe(false);
  });
});

describe('rowPhase()', () => {
  test('falls back to Pending', () => {
    expect(rowPhase(rowOf('t', 'Errored'))).toBe('Errored');
    expect(rowPhase(rowOf('t'))).toBe('Pending');
  });
});
