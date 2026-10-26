import { describe, expect, test } from 'vitest';

import { PromotionRequest } from '@ui/gen/api/v2/models';

import {
  behindCount,
  byWorstState,
  freightTally,
  stageRollouts,
  stateCounts,
  targetRows,
  worstState
} from './target-rows';
import { fleetStage, forStage, round, stage, target } from './test-fixtures';

describe('worstState()', () => {
  test('takes the worst of several Stages', () => {
    expect(worstState(['Healthy', 'Unhealthy', 'Behind'])).toBe('Unhealthy');
    expect(worstState(['Healthy', 'Behind'])).toBe('Behind');
    expect(worstState(['Unhealthy', 'Progressing'])).toBe('Progressing');
  });

  test('no governing Stage means never promoted', () => {
    expect(worstState([])).toBe('NotPromoted');
  });
});

describe('targetRows()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }], 'abc');
  const all = fleetStage('all-stage', [{}], 'abc');
  const classic = stage('classic');

  test('one row per Target with its governing Stages, both sorted by name', () => {
    const rows = targetRows(
      [
        target('us-west-2', { region: 'us' }, { 'us-stage': {}, 'all-stage': {} }),
        target('eu-central-1', { region: 'eu' }, { 'all-stage': {} })
      ],
      [us, classic, all]
    );
    expect(rows.map((r) => r.target.name)).toEqual(['eu-central-1', 'us-west-2']);
    expect(rows[0].stages.map((s) => s.stage.metadata?.name)).toEqual(['all-stage']);
    expect(rows[1].stages.map((s) => s.stage.metadata?.name)).toEqual(['all-stage', 'us-stage']);
  });

  test("which Stages govern a Target is the summary's to say", () => {
    // The page knows the Stage, but the summary does not name it, and names
    // one the page does not know: neither governs the row.
    const [row] = targetRows([target('t', { region: 'us' }, { unknown: forStage('abc') })], [us]);
    expect(row.stages).toEqual([]);
  });

  test('an ungoverned Target has never been promoted', () => {
    const [row] = targetRows([target('orphan')], [us]);
    expect(row.stages).toEqual([]);
    expect(row.state).toBe('NotPromoted');
    expect(row.behind).toBe(false);
    expect(row.freight).toEqual([]);
  });

  test('carries the Target health recorded for each Stage', () => {
    const t = target(
      't',
      { region: 'us' },
      { 'us-stage': forStage('abc', 'Unhealthy'), 'all-stage': forStage() }
    );
    const [row] = targetRows([t], [us, all]);
    const byStage = Object.fromEntries(row.stages.map((s) => [s.stage.metadata?.name, s.health]));
    expect(byStage['us-stage']?.status).toBe('Unhealthy');
    expect(byStage['all-stage']).toBeUndefined();
  });

  test.each([
    ['governed but never promoted to', forStage(), 'NotPromoted', false],
    ['on the latest Freight and healthy', forStage('abc', 'Healthy'), 'Healthy', false],
    ['healthy on older Freight', forStage('old', 'Healthy'), 'Behind', true],
    ['unhealthy', forStage('abc', 'Unhealthy'), 'Unhealthy', false],
    ['degraded', forStage('abc', 'Degraded'), 'Unhealthy', false],
    ['reported as unknown', forStage('abc', 'Unknown'), 'Unknown', false],
    ['unhealthy and behind', forStage('old', 'Unhealthy'), 'Unhealthy', true]
  ])('a Target %s is %s', (_name, summary, state, behind) => {
    const [row] = targetRows([target('t', { region: 'us' }, { 'us-stage': summary })], [us]);
    expect(row.state).toBe(state);
    expect(row.behind).toBe(behind);
  });

  // Targets carry no recorded health yet, so for most fleets the state comes
  // from how the Stage's latest Promotion to the Target went. These cases are
  // the ordinary ones.
  describe('with no health recorded', () => {
    const governed = (summary = forStage()) =>
      targetRows([target('t', { region: 'us' }, { 'us-stage': summary })], [us])[0];

    test('a Target the latest Promotion landed on is healthy, with nothing to compare', () => {
      const row = governed(forStage(undefined, undefined, undefined, 'Succeeded'));
      expect(row.state).toBe('Healthy');
      expect(row.behind).toBe(false);
      // Nothing to compare collections with, so the Promotion stands in for it.
      expect(row.stages[0].upToDate).toBe(true);
      expect(row.stages[0].promotion?.name).toBe('promotion');
    });

    test('a Target the latest Promotion failed on is unhealthy and behind', () => {
      const row = governed(forStage(undefined, undefined, undefined, 'Failed'));
      expect(row.state).toBe('Unhealthy');
      expect(row.behind).toBe(true);
    });

    test('a Target with Freight on it is healthy without a Promotion to go by', () => {
      expect(governed(forStage('abc')).state).toBe('Healthy');
    });

    test('a Target left on older Freight is behind, not healthy', () => {
      const row = governed(forStage('old'));
      expect(row.state).toBe('Behind');
      expect(row.behind).toBe(true);
    });

    test('a Target nothing has reached has not been promoted to', () => {
      expect(governed().state).toBe('NotPromoted');
    });

    test('recorded health outranks the Promotion it fell back to', () => {
      expect(governed(forStage('abc', 'Unhealthy', undefined, 'Succeeded')).state).toBe(
        'Unhealthy'
      );
    });
  });

  test("the Freight a Target runs comes from every Stage's collection", () => {
    const t = target(
      't',
      { region: 'us' },
      {
        'us-stage': forStage('abc', 'Healthy', 'freight-a'),
        'all-stage': forStage('old', 'Healthy', 'freight-b')
      }
    );
    expect(targetRows([t], [us, all])[0].freight).toEqual(['freight-b', 'freight-a']);
  });

  test('a Target the governing Stage is promoting to is Progressing', () => {
    const t = target(
      't',
      { region: 'us' },
      { 'us-stage': forStage('old', undefined, 'f', 'Running') }
    );
    const [row] = targetRows([t], [us]);
    expect(row.state).toBe('Progressing');
    expect(row.stages[0].phase).toBe('Running');
  });

  test('a Promotion yet to be picked up is Pending, and Progressing', () => {
    const t = target('t', { region: 'us' }, { 'us-stage': forStage('old', undefined, 'f', '') });
    const [row] = targetRows([t], [us]);
    expect(row.stages[0].phase).toBe('Pending');
    expect(row.state).toBe('Progressing');
  });

  test('a Promotion that has ended leaves the Target as its Freight says', () => {
    const t = target(
      't',
      { region: 'us' },
      { 'us-stage': forStage('abc', 'Healthy', 'f', 'Succeeded') }
    );
    expect(targetRows([t], [us])[0].state).toBe('Healthy');
  });
});

describe("targetRows() given the Stage's round", () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }], 'abc');
  const t = (summary = forStage('old', undefined, 'f', 'Succeeded')) =>
    target('t', { region: 'us' }, { 'us-stage': summary });

  test("reports the round's phase, not the latest Promotion's", () => {
    const running = round('us-stage', ['t'], { targets: [{ name: 't', phase: 'Running' }] });
    const [row] = targetRows([t()], [us], { 'us-stage': running });
    expect(row.stages[0].phase).toBe('Running');
    expect(row.state).toBe('Progressing');
  });

  test('a Target the round never named has no phase', () => {
    const [row] = targetRows([t()], [us], { 'us-stage': round('us-stage', ['other']) });
    expect(row.stages[0].phase).toBeUndefined();
    expect(row.state).toBe('Behind');
  });

  test('a round that never fanned out reaches no Target', () => {
    const blocked = round('us-stage', ['t'], {
      conditions: [
        { type: 'Ready', status: 'False', reason: 'EnterpriseOnlyFeature', message: 'nope' }
      ]
    } as Partial<PromotionRequest['status']>);
    const [row] = targetRows([t()], [us], { 'us-stage': blocked });
    expect(row.stages[0].phase).toBeUndefined();
  });
});

describe('stateCounts() and behindCount()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }], 'abc');
  const rows = targetRows(
    [
      target('a', { region: 'us' }, { 'us-stage': forStage('abc', 'Healthy') }),
      target('b', { region: 'us' }, { 'us-stage': forStage('old', 'Healthy') }),
      target('c', { region: 'us' }, { 'us-stage': forStage('old', 'Unhealthy') }),
      target('d', { region: 'us' })
    ],
    [us]
  );

  test('every state is tallied, including the empty ones', () => {
    expect(stateCounts(rows)).toEqual({
      Healthy: 1,
      Behind: 1,
      Progressing: 0,
      Unhealthy: 1,
      Unknown: 0,
      NotPromoted: 1
    });
  });

  test('the five tiles partition the rows', () => {
    const counts = stateCounts(rows);
    expect(
      counts.Healthy +
        counts.Behind +
        counts.Progressing +
        counts.Unhealthy +
        counts.Unknown +
        counts.NotPromoted
    ).toBe(rows.length);
  });

  test('behind cuts across the states', () => {
    expect(behindCount(rows)).toBe(2);
  });
});

describe('freightTally()', () => {
  const us = fleetStage('us-stage', [{}], 'abc');
  test('counts the Targets on each piece of Freight, commonest first', () => {
    const rows = targetRows(
      [
        target('a', {}, { 'us-stage': forStage('abc', 'Healthy', 'new') }),
        target('b', {}, { 'us-stage': forStage('old', 'Healthy', 'old') }),
        target('c', {}, { 'us-stage': forStage('abc', 'Healthy', 'new') }),
        target('d')
      ],
      [us]
    );
    expect(freightTally(rows)).toEqual([
      { name: 'new', count: 2 },
      { name: 'old', count: 1 }
    ]);
  });
});

describe('stageRollouts()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }], 'abc');
  const eu = fleetStage('eu-stage', [{ matchLabels: { region: 'eu' } }], 'abc');

  test('summarizes each Stage over the rows given', () => {
    const rows = targetRows(
      [
        target('a', { region: 'us' }, { 'us-stage': forStage('abc', 'Healthy', 'new') }),
        target('b', { region: 'us' }, { 'us-stage': forStage('old', 'Unhealthy') })
      ],
      [us, eu]
    );
    const [euRollout, usRollout] = stageRollouts(rows, [us, eu]);

    expect(usRollout.governed).toBe(2);
    expect(usRollout.onLatest).toBe(1);
    expect(usRollout.unhealthy).toBe(1);
    expect(usRollout.inFlight).toBe(0);
    expect(usRollout.counts.Healthy).toBe(1);

    // A Stage governing none of the rows still gets a rollout.
    expect(euRollout.governed).toBe(0);
  });

  test('names the Freight the Stage itself is on', () => {
    const current = fleetStage('cur', [{}]);
    current.status = { freightHistory: [{ id: 'abc', items: { 'Warehouse/w': { name: 'f1' } } }] };
    expect(stageRollouts([], [current])[0].freight).toEqual(['f1']);
  });
});

describe('byWorstState()', () => {
  const us = fleetStage('us-stage', [{}], 'abc');
  test('puts trouble first and ties break by name', () => {
    const rows = targetRows(
      [
        target('zed', {}, { 'us-stage': forStage('abc', 'Healthy') }),
        target('ann', {}, { 'us-stage': forStage('abc', 'Healthy') }),
        target('bob', {}, { 'us-stage': forStage('abc', 'Unhealthy') })
      ],
      [us]
    );
    expect(byWorstState(rows).map((row) => row.target.name)).toEqual(['bob', 'ann', 'zed']);
  });
});
