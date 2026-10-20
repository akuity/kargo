import { describe, expect, test } from 'vitest';

import { Promotion, PromotionRequest, TargetSummary } from '@ui/gen/api/v2/models';

import {
  ChildPromotion,
  childPhaseFilters,
  childPromotions,
  childrenSummary,
  conditionSummary,
  failedStepName,
  failuresByLabel,
  failuresByStep,
  firstFailure,
  matchesChild,
  stepStatuses
} from './round-detail';

const request = (partial: Partial<PromotionRequest>): PromotionRequest =>
  ({ spec: { stage: 'prod-us', freight: 'abc', targets: [] }, ...partial }) as PromotionRequest;

const promotion = (name: string, partial: Partial<Promotion> = {}): Promotion =>
  ({ metadata: { name }, spec: { stage: 'prod-us', steps: [] }, ...partial }) as Promotion;

const target = (name: string, labels: Record<string, string> = {}): TargetSummary => ({
  name,
  labels
});

// child builds a ChildPromotion directly, for the tallies that only care about
// a phase and some labels.
const child = (
  partial: Partial<ChildPromotion> & { target: string; phase: ChildPromotion['phase'] }
): ChildPromotion => ({ labels: {}, ...partial });

describe('childPromotions()', () => {
  const base = request({
    spec: {
      stage: 'prod-us',
      freight: 'abc',
      targets: [{ name: 'east-1' }, { name: 'east-2' }, { name: 'west-1' }]
    },
    status: {
      targets: [
        { name: 'east-1', promotion: 'promo-east-1', phase: 'Succeeded' },
        { name: 'east-2', promotion: 'promo-east-2', phase: 'Running' }
      ]
    }
  });

  test('pairs each Target with its child Promotion and labels', () => {
    const children = childPromotions(
      base,
      [promotion('promo-east-1', { status: { phase: 'Succeeded' } })],
      [target('east-1', { tier: 'primary' })]
    );
    const east1 = children.find((entry) => entry.target === 'east-1');
    expect(east1?.promotion?.metadata?.name).toBe('promo-east-1');
    expect(east1?.labels).toEqual({ tier: 'primary' });
  });

  test("the Promotion's own phase wins over the round's snapshot of it", () => {
    const children = childPromotions(base, [
      promotion('promo-east-2', { status: { phase: 'Failed' } })
    ]);
    expect(children.find((entry) => entry.target === 'east-2')?.phase).toBe('Failed');
  });

  test('a Target the round has not reached yet is Pending', () => {
    const children = childPromotions(base);
    expect(children.find((entry) => entry.target === 'west-1')?.phase).toBe('Pending');
  });

  test('orders trouble first, then what is moving, then by name', () => {
    const children = childPromotions(base, [
      promotion('promo-east-1', { status: { phase: 'Succeeded' } }),
      promotion('promo-east-2', { status: { phase: 'Failed' } })
    ]);
    expect(children.map((entry) => entry.target)).toEqual(['east-2', 'west-1', 'east-1']);
  });
});

describe('childrenSummary()', () => {
  test('tallies the children by phase', () => {
    expect(
      childrenSummary([
        child({ target: 'a', phase: 'Succeeded' }),
        child({ target: 'b', phase: 'Succeeded' }),
        child({ target: 'c', phase: 'Failed' })
      ])
    ).toEqual({ succeeded: 2, failed: 1 });
  });

  test('a round with no children tallies nothing', () => {
    expect(childrenSummary([])).toEqual({});
  });
});

describe('stepStatuses()', () => {
  test('one entry per step, unreached steps left blank', () => {
    expect(
      stepStatuses(
        promotion('p', {
          spec: { stage: 'prod-us', steps: [{ uses: 'a' }, { uses: 'b' }, { uses: 'c' }] },
          status: { stepExecutionMetadata: [{ status: 'Succeeded' }, { status: 'Failed' }] }
        } as Partial<Promotion>)
      )
    ).toEqual(['Succeeded', 'Failed', undefined]);
  });

  test('a Promotion with no steps has none', () => {
    expect(stepStatuses(promotion('p'))).toEqual([]);
    expect(stepStatuses(undefined)).toEqual([]);
  });
});

describe('failedStepName()', () => {
  const withSteps = (...metadata: { status: string; alias?: string }[]) =>
    promotion('p', { status: { stepExecutionMetadata: metadata } } as Partial<Promotion>);

  test('names the failed step by its alias', () => {
    expect(
      failedStepName(
        withSteps(
          { status: 'Succeeded', alias: 'task-0' },
          { status: 'Failed', alias: 'task-1::fail' }
        )
      )
    ).toBe('task-1::fail');
  });

  test('falls back to the step position', () => {
    expect(failedStepName(withSteps({ status: 'Succeeded' }, { status: 'Errored' }))).toBe(
      'step 2'
    );
  });

  test('a Promotion that did not fail names no step', () => {
    expect(failedStepName(withSteps({ status: 'Succeeded' }))).toBeUndefined();
    expect(failedStepName(undefined)).toBeUndefined();
  });
});

describe('failuresByStep()', () => {
  const failedOn = (name: string, alias: string) =>
    child({
      target: name,
      phase: 'Failed',
      promotion: promotion(name, {
        status: { stepExecutionMetadata: [{ status: 'Failed', alias }] }
      } as Partial<Promotion>)
    });

  test('counts the children that stopped on each step, worst first', () => {
    expect(
      failuresByStep([
        failedOn('a', 'task-1::fail'),
        failedOn('b', 'task-1::fail'),
        failedOn('c', 'wait'),
        child({ target: 'd', phase: 'Succeeded' })
      ])
    ).toEqual([
      { key: 'task-1::fail', failed: 2 },
      { key: 'wait', failed: 1 }
    ]);
  });

  test('a round with nothing wrong has nothing to show', () => {
    expect(failuresByStep([child({ target: 'a', phase: 'Succeeded' })])).toEqual([]);
  });
});

describe('failuresByLabel()', () => {
  // The mockup round: 136 Targets split evenly by tier, 94 of them on AWS, and
  // 6 failures -- five secondary, one primary, four of them on AWS.
  const succeeded = (count: number, prefix: string, labels: Record<string, string>) =>
    Array.from({ length: count }, (_, i) =>
      child({ target: `${prefix}${i}`, phase: 'Succeeded', labels })
    );
  const children = [
    child({ target: 's0', phase: 'Failed', labels: { tier: 'secondary', cloud: 'aws' } }),
    child({ target: 's1', phase: 'Failed', labels: { tier: 'secondary', cloud: 'aws' } }),
    child({ target: 's2', phase: 'Failed', labels: { tier: 'secondary', cloud: 'aws' } }),
    child({ target: 's3', phase: 'Failed', labels: { tier: 'secondary', cloud: 'azure' } }),
    child({ target: 's4', phase: 'Failed', labels: { tier: 'secondary', cloud: 'gcp' } }),
    child({ target: 'p0', phase: 'Failed', labels: { tier: 'primary', cloud: 'aws' } }),
    ...succeeded(45, 'sa', { tier: 'secondary', cloud: 'aws' }),
    ...succeeded(18, 'sg', { tier: 'secondary', cloud: 'gcp' }),
    ...succeeded(45, 'pa', { tier: 'primary', cloud: 'aws' }),
    ...succeeded(22, 'pg', { tier: 'primary', cloud: 'gcp' })
  ];

  test('counts failures over how many Targets carry the label', () => {
    expect(failuresByLabel(children)).toEqual([
      { key: 'tier=secondary', failed: 5, total: 68 },
      { key: 'tier=primary', failed: 1, total: 68 },
      { key: 'cloud=aws', failed: 4, total: 94 },
      // azure and gcp tie on one failure each, so they fall back to name order.
      { key: 'cloud=azure', failed: 1, total: 1 },
      { key: 'cloud=gcp', failed: 1, total: 41 }
    ]);
  });

  test('values of one key stay together, keys ordered by their worst value', () => {
    const skewed = [
      child({ target: 'a', phase: 'Failed', labels: { tier: 'secondary', cloud: 'aws' } }),
      child({ target: 'b', phase: 'Failed', labels: { tier: 'secondary', cloud: 'gcp' } }),
      child({ target: 'c', phase: 'Failed', labels: { tier: 'primary', cloud: 'aws' } })
    ];
    // tier's worst value has 2 failures against cloud's 2 as well, so the keys
    // tie and fall back to name order -- but neither key is ever interleaved.
    expect(failuresByLabel(skewed).map((tally) => tally.key)).toEqual([
      'cloud=aws',
      'cloud=gcp',
      'tier=secondary',
      'tier=primary'
    ]);
  });

  test('only labels with a failure against them are listed', () => {
    expect(
      failuresByLabel([child({ target: 'a', phase: 'Succeeded', labels: { tier: 'x' } })])
    ).toEqual([]);
  });

  test('the list is capped', () => {
    const many = Array.from({ length: 10 }, (_, i) =>
      child({ target: `t${i}`, phase: 'Failed', labels: { [`key${i}`]: 'v' } })
    );
    expect(failuresByLabel(many, 3)).toHaveLength(3);
  });
});

describe('firstFailure()', () => {
  const failedAt = (name: string, finishedAt?: string) =>
    child({
      target: name,
      phase: 'Failed',
      promotion: promotion(name, { status: { finishedAt } } as Partial<Promotion>)
    });

  test('the failure that finished earliest', () => {
    expect(
      firstFailure([
        failedAt('late', '2026-10-13T03:18:15Z'),
        failedAt('early', '2026-10-13T03:14:57Z'),
        child({ target: 'ok', phase: 'Succeeded' })
      ])?.target
    ).toBe('early');
  });

  test('a failure that cannot be placed in time yields to one that can', () => {
    expect(
      firstFailure([failedAt('unknown'), failedAt('known', '2026-10-13T03:14:57Z')])?.target
    ).toBe('known');
  });

  test('a round with nothing wrong has no first failure', () => {
    expect(firstFailure([child({ target: 'a', phase: 'Succeeded' })])).toBeUndefined();
  });
});

describe('conditionSummary()', () => {
  test('reports the Ready condition with its reason', () => {
    expect(
      conditionSummary(
        request({
          status: {
            conditions: [{ type: 'Ready', status: 'False', reason: 'PromotionsFailed' }]
          }
        } as Partial<PromotionRequest>)
      )
    ).toBe('Ready=False · PromotionsFailed');
  });

  test('a condition with no reason reports only its status', () => {
    expect(
      conditionSummary(
        request({
          status: { conditions: [{ type: 'Ready', status: 'True' }] }
        } as Partial<PromotionRequest>)
      )
    ).toBe('Ready=True');
  });

  test('a round with no Ready condition says nothing', () => {
    expect(conditionSummary(request({ status: {} }))).toBeUndefined();
    expect(conditionSummary(undefined)).toBeUndefined();
  });
});

describe('childPhaseFilters()', () => {
  test('the mockup round: All, then every phase, leading with what broke', () => {
    expect(
      childPhaseFilters([
        child({ target: 'a', phase: 'Failed' }),
        child({ target: 'b', phase: 'Succeeded' }),
        child({ target: 'c', phase: 'Succeeded' })
      ])
    ).toEqual([
      { count: 3 },
      { phase: 'Failed', count: 1 },
      { phase: 'Errored', count: 0 },
      { phase: 'Running', count: 0 },
      { phase: 'Pending', count: 0 },
      { phase: 'Succeeded', count: 2 }
    ]);
  });

  test('Aborted appears only once something was cancelled', () => {
    const filters = childPhaseFilters([child({ target: 'a', phase: 'Aborted' })]);
    expect(filters.map((filter) => filter.phase)).toContain('Aborted');
    expect(childPhaseFilters([]).map((filter) => filter.phase)).not.toContain('Aborted');
  });
});

describe('matchesChild()', () => {
  const entry = child({ target: 'prod-us-east-3', phase: 'Failed', labels: { tier: 'secondary' } });

  test('matches the Target name and its labels as key=value', () => {
    expect(matchesChild(entry, 'east')).toBe(true);
    expect(matchesChild(entry, 'tier=secondary')).toBe(true);
    expect(matchesChild(entry, 'TIER=SECONDARY')).toBe(true);
  });

  test('an empty needle matches everything', () => {
    expect(matchesChild(entry, '   ')).toBe(true);
  });

  test('anything else does not match', () => {
    expect(matchesChild(entry, 'west')).toBe(false);
  });
});
