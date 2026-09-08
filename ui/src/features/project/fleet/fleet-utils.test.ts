import { describe, expect, test } from 'vitest';

import { PromotionRequest, Stage, Target, V1LabelSelector } from '@ui/gen/api/v2/models';

import {
  GROUP_BY_STAGE,
  UNGOVERNED,
  UNLABELED,
  fleetRows,
  groupRows,
  groupSeverity,
  labelKeys,
  rowPhaseCounts,
  matchesLabelSelector,
  phaseCounts,
  requestForStage,
  stageGovernsTarget,
  targetAwareStages,
  targetStageCell
} from './fleet-utils';

const target = (name: string, labels?: Record<string, string>): Target => ({
  metadata: { name, labels }
});

const stage = (name: string, partial: Partial<Stage> = {}): Stage => ({
  metadata: { name },
  ...partial
});

const fleetStage = (name: string, selectors: V1LabelSelector[]) =>
  stage(name, { spec: { requestedFreight: [], targets: { selectors } } } as Partial<Stage>);

const request = (
  name: string,
  stageName: string,
  targets: string[],
  status?: PromotionRequest['status']
): PromotionRequest => ({
  metadata: { name },
  spec: { stage: stageName, freight: 'f1', targets: targets.map((t) => ({ name: t })) },
  status
});

describe('matchesLabelSelector()', () => {
  test('an empty selector matches everything', () => {
    expect(matchesLabelSelector({}, {})).toBe(true);
    expect(matchesLabelSelector(undefined, { a: 'b' })).toBe(true);
  });

  test('matchLabels are ANDed', () => {
    const selector = { matchLabels: { region: 'us', tier: 'prod' } };
    expect(matchesLabelSelector(selector, { region: 'us', tier: 'prod' })).toBe(true);
    expect(matchesLabelSelector(selector, { region: 'us' })).toBe(false);
  });

  test.each([
    ['In', ['us', 'eu'], { region: 'us' }, true],
    ['In', ['us', 'eu'], { region: 'ap' }, false],
    ['In', ['us'], {}, false],
    ['NotIn', ['us'], { region: 'eu' }, true],
    ['NotIn', ['us'], {}, true],
    ['NotIn', ['us'], { region: 'us' }, false],
    ['Exists', undefined, { region: 'x' }, true],
    ['Exists', undefined, {}, false],
    ['DoesNotExist', undefined, {}, true],
    ['DoesNotExist', undefined, { region: 'x' }, false],
    ['Bogus', undefined, { region: 'x' }, false]
  ])('%s %j against %j -> %s', (operator, values, labels, expected) => {
    expect(
      matchesLabelSelector(
        { matchExpressions: [{ key: 'region', operator, values }] },
        labels as Record<string, string>
      )
    ).toBe(expected);
  });
});

describe('stageGovernsTarget()', () => {
  test('a classic Stage governs nothing', () => {
    expect(stageGovernsTarget(stage('classic'), target('t', { region: 'us' }))).toBe(false);
  });

  test('an empty selector list governs nothing', () => {
    expect(stageGovernsTarget(fleetStage('fleet', []), target('t', { region: 'us' }))).toBe(false);
  });

  test('selectors are a union', () => {
    const fleet = fleetStage('fleet', [
      { matchLabels: { region: 'us' } },
      { matchLabels: { region: 'eu' } }
    ]);
    expect(stageGovernsTarget(fleet, target('a', { region: 'us' }))).toBe(true);
    expect(stageGovernsTarget(fleet, target('b', { region: 'eu' }))).toBe(true);
    expect(stageGovernsTarget(fleet, target('c', { region: 'ap' }))).toBe(false);
  });
});

describe('targetAwareStages()', () => {
  test('keeps only target-aware Stages, sorted by name', () => {
    const result = targetAwareStages([
      fleetStage('zeta', []),
      stage('classic'),
      fleetStage('alpha', [])
    ]);
    expect(result.map((s) => s.metadata?.name)).toEqual(['alpha', 'zeta']);
  });
});

describe('labelKeys()', () => {
  test('collects distinct keys, sorted, minus Kargo-internal ones', () => {
    expect(
      labelKeys([
        target('a', { region: 'us', 'kargo.akuity.io/shard': 'x' }),
        target('b', { env: 'prod', region: 'eu' }),
        target('c')
      ])
    ).toEqual(['env', 'region']);
  });
});

describe('fleetRows()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }]);
  const all = fleetStage('all-stage', [{}]);
  const targets = [
    target('us-west-2', { region: 'us' }),
    target('eu-central-1', { region: 'eu' }),
    target('us-east-1', { region: 'us' })
  ];

  test('one row per governing Stage, sorted by Target then Stage', () => {
    const rows = fleetRows(targets, [us, all], []);
    expect(rows.map((r) => r.key)).toEqual([
      'eu-central-1/all-stage',
      'us-east-1/all-stage',
      'us-east-1/us-stage',
      'us-west-2/all-stage',
      'us-west-2/us-stage'
    ]);
  });

  test('an ungoverned Target still gets a row, with no Stage', () => {
    const rows = fleetRows(targets, [us], []);
    const orphan = rows.find((r) => r.target.metadata?.name === 'eu-central-1');
    expect(orphan?.stage).toBeUndefined();
    expect(orphan?.cell).toEqual({ governed: false, included: false });
  });

  test('no Targets yields no rows', () => {
    expect(fleetRows([], [us], [])).toEqual([]);
  });
});

describe('groupRows()', () => {
  const us = fleetStage('us-stage', [{ matchLabels: { region: 'us' } }]);
  const rows = fleetRows(
    [
      target('us-west-2', { region: 'us' }),
      target('eu-central-1', { region: 'eu' }),
      target('orphan'),
      target('us-east-1', { region: 'us' })
    ],
    [us],
    []
  );

  test('without a key, one group', () => {
    const groups = groupRows(rows);
    expect(groups).toHaveLength(1);
    expect(groups[0].rows).toHaveLength(4);
  });

  test('by Stage, ungoverned last', () => {
    const groups = groupRows(rows, GROUP_BY_STAGE);
    expect(groups.map((g) => g.value)).toEqual(['us-stage', UNGOVERNED]);
    expect(groups[0].rows.map((r) => r.target.metadata?.name)).toEqual(['us-east-1', 'us-west-2']);
    expect(groups[1].rows.map((r) => r.target.metadata?.name)).toEqual(['eu-central-1', 'orphan']);
  });

  test('by label, unlabeled last', () => {
    const groups = groupRows(rows, 'region');
    expect(groups.map((g) => g.value)).toEqual(['eu', 'us', UNLABELED]);
  });

  test('no rows yields no groups', () => {
    expect(groupRows([], GROUP_BY_STAGE)).toEqual([]);
  });

  test('groups with trouble sort first, then active, then settled', () => {
    const a = fleetStage('a-ok', [{ matchLabels: { s: 'a' } }]);
    const b = fleetStage('b-running', [{ matchLabels: { s: 'b' } }]);
    const c = fleetStage('c-failed', [{ matchLabels: { s: 'c' } }]);
    const requests = [
      request('a-ok.01', 'a-ok', ['ta'], {
        phase: 'Succeeded',
        targets: [{ name: 'ta', promotion: 'p', phase: 'Succeeded' }]
      }),
      request('b-running.01', 'b-running', ['tb'], { phase: 'Running' }),
      request('c-failed.01', 'c-failed', ['tc'], {
        phase: 'Errored',
        targets: [{ name: 'tc', promotion: 'p', phase: 'Errored' }]
      })
    ];
    const groups = groupRows(
      fleetRows(
        [target('ta', { s: 'a' }), target('tb', { s: 'b' }), target('tc', { s: 'c' })],
        [a, b, c],
        requests
      ),
      GROUP_BY_STAGE
    );
    expect(groups.map((g) => g.value)).toEqual(['c-failed', 'b-running', 'a-ok']);
    expect(groups.map(groupSeverity)).toEqual([0, 1, 2]);
  });
});

describe('rowPhaseCounts()', () => {
  test('counts governed rows by phase, defaulting to Pending', () => {
    const s = fleetStage('s', [{ matchLabels: { g: 'y' } }]);
    const rows = fleetRows(
      [target('t1', { g: 'y' }), target('t2', { g: 'y' }), target('orphan')],
      [s],
      [
        request('s.01', 's', ['t1', 't2'], {
          phase: 'Running',
          targets: [{ name: 't1', promotion: 'p', phase: 'Succeeded' }]
        })
      ]
    );
    expect(rowPhaseCounts(rows)).toEqual({ total: 2, byPhase: { Succeeded: 1, Pending: 1 } });
  });
});

describe('requestForStage()', () => {
  const older = request('fleet.01a.x', 'fleet', ['t']);
  const newer = request('fleet.01b.x', 'fleet', ['t']);
  const other = request('other.01c.x', 'other', ['t']);
  const requests = [older, newer, other];

  test('prefers the current request', () => {
    const s = stage('fleet', {
      status: {
        currentPromotionRequest: { name: older.metadata!.name },
        lastPromotionRequest: { name: newer.metadata!.name }
      }
    } as Partial<Stage>);
    expect(requestForStage(s, requests)).toBe(older);
  });

  test('falls back to the last request', () => {
    const s = stage('fleet', {
      status: { lastPromotionRequest: { name: older.metadata!.name } }
    } as Partial<Stage>);
    expect(requestForStage(s, requests)).toBe(older);
  });

  test('falls back to the newest request for the Stage', () => {
    expect(requestForStage(stage('fleet'), requests)).toBe(newer);
  });

  test('ignores other Stages and returns undefined when there are none', () => {
    expect(requestForStage(stage('lonely'), requests)).toBeUndefined();
  });
});

describe('targetStageCell()', () => {
  const fleet = fleetStage('fleet', [{ matchLabels: { region: 'us' } }]);
  const usEast = target('us-east-1', { region: 'us' });
  const euWest = target('eu-west-1', { region: 'eu' });

  test('ungoverned Target with no request', () => {
    expect(targetStageCell(fleet, euWest, [])).toEqual({
      governed: false,
      request: undefined,
      included: false
    });
  });

  test('governed Target the Stage has never promoted to', () => {
    expect(targetStageCell(fleet, usEast, [])).toMatchObject({ governed: true, included: false });
  });

  test('governed Target that joined after the round was created', () => {
    const round = request('fleet.01a.x', 'fleet', ['us-west-2']);
    const cell = targetStageCell(fleet, usEast, [round]);
    expect(cell).toMatchObject({ governed: true, request: round, included: false });
    expect(cell.phase).toBeUndefined();
  });

  test('reads the child Promotion phase from the round', () => {
    const round = request('fleet.01a.x', 'fleet', ['us-east-1'], {
      phase: 'Running',
      targets: [{ name: 'us-east-1', promotion: 'fleet.us-east-1.01a.x', phase: 'Succeeded' }]
    });
    expect(targetStageCell(fleet, usEast, [round])).toMatchObject({
      included: true,
      promotion: 'fleet.us-east-1.01a.x',
      phase: 'Succeeded'
    });
  });

  test('is Pending while a live round has not reached the Target', () => {
    const round = request('fleet.01a.x', 'fleet', ['us-east-1'], { phase: 'Running' });
    expect(targetStageCell(fleet, usEast, [round]).phase).toBe('Pending');
  });

  test('takes the round phase when the round ended before creating a child', () => {
    const round = request('fleet.01a.x', 'fleet', ['us-east-1'], { phase: 'Errored' });
    expect(targetStageCell(fleet, usEast, [round]).phase).toBe('Errored');
  });
});

describe('phaseCounts()', () => {
  test('tallies governed Targets only, defaulting to Pending', () => {
    const fleet = fleetStage('fleet', [{ matchLabels: { region: 'us' } }]);
    const round = request('fleet.01a.x', 'fleet', ['us-east-1', 'us-west-2'], {
      phase: 'Running',
      targets: [{ name: 'us-east-1', promotion: 'p', phase: 'Succeeded' }]
    });
    const counts = phaseCounts(
      fleet,
      [
        target('us-east-1', { region: 'us' }),
        target('us-west-2', { region: 'us' }),
        target('eu-west-1', { region: 'eu' })
      ],
      [round]
    );
    expect(counts).toEqual({ total: 2, byPhase: { Succeeded: 1, Pending: 1 } });
  });
});
