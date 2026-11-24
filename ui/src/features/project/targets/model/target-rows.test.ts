import { describe, expect, test } from 'vitest';

import { PromotionRequest, Stage } from '@ui/gen/api/v2/models';

import {
  Rounds,
  aheadFreight,
  behindCount,
  byWorstState,
  freightTally,
  scopeToStage,
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
      [target('us-west-2', { region: 'us' }), target('eu-central-1', { region: 'eu' })],
      [us, classic, all]
    );
    expect(rows.map((r) => r.target.metadata?.name)).toEqual(['eu-central-1', 'us-west-2']);
    expect(rows[0].stages.map((s) => s.stage.metadata?.name)).toEqual(['all-stage']);
    expect(rows[1].stages.map((s) => s.stage.metadata?.name)).toEqual(['all-stage', 'us-stage']);
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
      { status: { stages: { 'us-stage': forStage('abc', 'Unhealthy') } } }
    );
    const [row] = targetRows([t], [us, all]);
    const byStage = Object.fromEntries(row.stages.map((s) => [s.stage.metadata?.name, s.health]));
    expect(byStage['us-stage']?.status).toBe('Unhealthy');
    expect(byStage['all-stage']).toBeUndefined();
  });

  test.each([
    ['governed but never promoted to', undefined, 'NotPromoted', false],
    ['on the latest Freight and healthy', forStage('abc', 'Healthy'), 'Healthy', false],
    ['healthy on older Freight', forStage('old', 'Healthy'), 'Behind', true],
    ['unhealthy', forStage('abc', 'Unhealthy'), 'Unhealthy', false],
    ['degraded', forStage('abc', 'Degraded'), 'Unhealthy', false],
    ['reported as unknown', forStage('abc', 'Unknown'), 'Unknown', false],
    ['unhealthy and behind', forStage('old', 'Unhealthy'), 'Unhealthy', true]
  ])('a Target %s is %s', (_name, status, state, behind) => {
    const t = target('t', { region: 'us' }, { status: { stages: { 'us-stage': status } } });
    const [row] = targetRows([t], [us]);
    expect(row.state).toBe(state);
    expect(row.behind).toBe(behind);
  });

  // Targets carry no recorded health yet, so for most fleets the state comes
  // from how the Stage's latest round went. These cases are the ordinary ones.
  describe('with no health recorded', () => {
    const succeeded = (name: string) =>
      round('us-stage', [name], {
        phase: 'Succeeded',
        targets: [{ name, phase: 'Succeeded' }]
      });

    test('a Target the latest round landed on is healthy, even with no status', () => {
      const t = target('t', { region: 'us' });
      const [row] = targetRows([t], [us], { 'us-stage': succeeded('t') });
      expect(row.state).toBe('Healthy');
      expect(row.behind).toBe(false);
      // Nothing to compare collections with, so the round stands in for it.
      expect(row.stages[0].upToDate).toBe(true);
    });

    test('a Target the latest round failed on is unhealthy and behind', () => {
      const t = target('t', { region: 'us' });
      const failed = round('us-stage', ['t'], {
        phase: 'Failed',
        targets: [{ name: 't', phase: 'Failed' }]
      });
      const [row] = targetRows([t], [us], { 'us-stage': failed });
      expect(row.state).toBe('Unhealthy');
      expect(row.behind).toBe(true);
    });

    test('a Target with Freight on it is healthy without a round at all', () => {
      const t = target(
        't',
        { region: 'us' },
        { status: { stages: { 'us-stage': forStage('abc') } } }
      );
      expect(targetRows([t], [us])[0].state).toBe('Healthy');
    });

    test('a Target left on older Freight is behind, not healthy', () => {
      const t = target(
        't',
        { region: 'us' },
        { status: { stages: { 'us-stage': forStage('old') } } }
      );
      const [row] = targetRows([t], [us]);
      expect(row.state).toBe('Behind');
      expect(row.behind).toBe(true);
    });

    test('a Target no round and no Freight reaches has not been promoted to', () => {
      const t = target('t', { region: 'us' });
      expect(targetRows([t], [us], { 'us-stage': round('us-stage', ['other']) })[0].state).toBe(
        'NotPromoted'
      );
    });

    test('recorded health outranks the round it fell back to', () => {
      const t = target(
        't',
        { region: 'us' },
        { status: { stages: { 'us-stage': forStage('abc', 'Unhealthy') } } }
      );
      expect(targetRows([t], [us], { 'us-stage': succeeded('t') })[0].state).toBe('Unhealthy');
    });
  });

  test("the Freight a Target runs comes from every Stage's collection", () => {
    const t = target(
      't',
      { region: 'us' },
      {
        status: {
          stages: {
            'us-stage': forStage('abc', 'Healthy', 'freight-a'),
            'all-stage': forStage('old', 'Healthy', 'freight-b')
          }
        }
      }
    );
    expect(targetRows([t], [us, all])[0].freight).toEqual(['freight-b', 'freight-a']);
  });

  test('a Target the governing Stage is promoting to is Progressing', () => {
    const t = target(
      't',
      { region: 'us' },
      { status: { stages: { 'us-stage': forStage('old') } } }
    );
    const rounds: Rounds = {
      'us-stage': round('us-stage', ['t'], { targets: [{ name: 't', phase: 'Running' }] })
    };
    expect(targetRows([t], [us], rounds)[0].state).toBe('Progressing');
  });

  test('a Target the round has yet to reach is Progressing', () => {
    const t = target(
      't',
      { region: 'us' },
      { status: { stages: { 'us-stage': forStage('old') } } }
    );
    expect(targetRows([t], [us], { 'us-stage': round('us-stage', ['t']) })[0].state).toBe(
      'Progressing'
    );
  });

  test('a round that has ended leaves its Targets as their status says', () => {
    const t = target(
      't',
      { region: 'us' },
      { status: { stages: { 'us-stage': forStage('abc', 'Healthy') } } }
    );
    const rounds: Rounds = {
      'us-stage': round('us-stage', ['t'], {
        phase: 'Succeeded',
        targets: [{ name: 't', phase: 'Succeeded' }]
      })
    };
    expect(targetRows([t], [us], rounds)[0].state).toBe('Healthy');
  });

  test('a round that never fanned out leaves its Targets alone', () => {
    const t = target(
      't',
      { region: 'us' },
      { status: { stages: { 'us-stage': forStage('old') } } }
    );
    const blocked = round('us-stage', ['t'], {
      conditions: [
        { type: 'Ready', status: 'False', reason: 'EnterpriseOnlyFeature', message: 'nope' }
      ]
    } as Partial<PromotionRequest['status']>);
    expect(targetRows([t], [us], { 'us-stage': blocked })[0].state).toBe('Behind');
  });

  test('a Target the round never named is left alone', () => {
    const t = target(
      't',
      { region: 'us' },
      { status: { stages: { 'us-stage': forStage('abc', 'Healthy') } } }
    );
    expect(targetRows([t], [us], { 'us-stage': round('us-stage', ['other']) })[0].state).toBe(
      'Healthy'
    );
  });
});

describe('stateCounts() and behindCount()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }], 'abc');
  const rows = targetRows(
    [
      target(
        'a',
        { region: 'us' },
        { status: { stages: { 'us-stage': forStage('abc', 'Healthy') } } }
      ),
      target(
        'b',
        { region: 'us' },
        { status: { stages: { 'us-stage': forStage('old', 'Healthy') } } }
      ),
      target(
        'c',
        { region: 'us' },
        { status: { stages: { 'us-stage': forStage('old', 'Unhealthy') } } }
      ),
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
        target('a', {}, { status: { stages: { 'us-stage': forStage('abc', 'Healthy', 'new') } } }),
        target('b', {}, { status: { stages: { 'us-stage': forStage('old', 'Healthy', 'old') } } }),
        target('c', {}, { status: { stages: { 'us-stage': forStage('abc', 'Healthy', 'new') } } }),
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

describe('scopeToStage()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }], 'new');
  const all = fleetStage('all-stage', [{}], 'new');

  test('reads the row as one Stage sees it', () => {
    const [row] = targetRows(
      [
        target(
          'a',
          { region: 'us' },
          {
            status: {
              stages: {
                'all-stage': forStage('old', 'Healthy', 'f-old'),
                'us-stage': forStage('new', 'Unhealthy', 'f-new')
              }
            }
          }
        )
      ],
      [us, all]
    );
    expect(row.state).toBe('Unhealthy');
    expect(row.behind).toBe(true);

    const [allStage, usStage] = row.stages;
    const scoped = scopeToStage(row, allStage);
    expect(scoped.scopedTo).toBe(allStage);
    expect(scoped.state).toBe('Behind');
    expect(scoped.behind).toBe(true);
    expect(scoped.freight).toEqual(['f-old']);

    const other = scopeToStage(row, usStage);
    expect(other.state).toBe('Unhealthy');
    expect(other.behind).toBe(false);
    expect(other.freight).toEqual(['f-new']);
    expect(other.target).toBe(row.target);
  });

  test('leaves the governing Stages whole, so the detail views keep them', () => {
    const [row] = targetRows([target('a', { region: 'us' })], [us, all]);
    expect(scopeToStage(row, row.stages[0]).stages).toBe(row.stages);
  });

  // Keeping every governing Stage on the row means anything reading them has
  // to say which Stage it is answering for. aheadFreight is the one that does.
  test('aheadFreight answers for the scoped Stage alone', () => {
    // The Stages need Freight of their own, since that is what a "behind" tag
    // names: what the Stage has moved on to.
    const onLatest = (stage: Stage, freight: string): Stage => ({
      ...stage,
      status: { freightHistory: [{ id: 'new', items: { 'Warehouse/w': { name: freight } } }] }
    });
    const [row] = targetRows(
      [
        target(
          'a',
          { region: 'us' },
          {
            status: {
              stages: {
                // Behind: the Stage is on 'new', the Target still on 'old'.
                'all-stage': forStage('old', 'Healthy', 'f-old'),
                // Up to date.
                'us-stage': forStage('new', 'Healthy', 'f-new')
              }
            }
          }
        )
      ],
      [onLatest(us, 'us-latest'), onLatest(all, 'all-latest')]
    );
    const [allStage, usStage] = row.stages;

    // Unscoped, the row speaks for every Stage, and all-stage is behind.
    expect(aheadFreight(row)).toEqual(['all-latest']);
    // Scoped to all-stage, which is the Stage that is behind.
    expect(aheadFreight(scopeToStage(row, allStage))).toEqual(['all-latest']);
    // Scoped to us-stage, which is up to date: no "behind" tag belongs on a
    // row this very view calls Healthy.
    expect(aheadFreight(scopeToStage(row, usStage))).toEqual([]);
  });
});

describe('byWorstState()', () => {
  const us = fleetStage('us-stage', [{}], 'abc');
  test('puts trouble first and ties break by name', () => {
    const rows = targetRows(
      [
        target('zed', {}, { status: { stages: { 'us-stage': forStage('abc', 'Healthy') } } }),
        target('ann', {}, { status: { stages: { 'us-stage': forStage('abc', 'Healthy') } } }),
        target('bob', {}, { status: { stages: { 'us-stage': forStage('abc', 'Unhealthy') } } })
      ],
      [us]
    );
    expect(byWorstState(rows).map((row) => row.target.metadata?.name)).toEqual([
      'bob',
      'ann',
      'zed'
    ]);
  });
});
