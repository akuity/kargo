import { describe, expect, test } from 'vitest';

import { PromotionRollout, Stage } from '@ui/gen/api/v2/models';

import {
  getCurrentPromotion,
  getCurrentPromotionRef,
  getLastPromotionDate,
  getLastPromotionRef,
  isStageTargetAware
} from './stage-meta-utils';

const stage = (partial: Partial<Stage>): Stage =>
  ({
    metadata: { namespace: 'my-project', name: 'prod' },
    ...partial
  }) as Stage;

// classic promotes through Promotions: no targets block.
const classic = (status: Stage['status']) => stage({ status });

// targetAware promotes through rounds of fan-out -- PromotionRollouts or
// PromotionRequests: a targets block is present, whether or not it currently
// matches anything.
const targetAware = (status: Stage['status']) => stage({ spec: { targets: {} }, status });

const promoting = [{ type: 'Promoting', status: 'True' }] as NonNullable<
  Stage['status']
>['conditions'];

describe('isStageTargetAware()', () => {
  test('a Stage with a targets block is target-aware', () => {
    expect(isStageTargetAware(targetAware({}))).toBe(true);
  });

  test('a Stage without one is not', () => {
    expect(isStageTargetAware(classic({}))).toBe(false);
    expect(isStageTargetAware(undefined)).toBe(false);
  });
});

describe('getCurrentPromotionRef()', () => {
  test('classic Stage: undefined when nothing is running', () => {
    expect(getCurrentPromotionRef(classic({}))).toBeUndefined();
  });

  test('classic Stage: the current Promotion, linking to its page', () => {
    expect(
      getCurrentPromotionRef(
        classic({
          currentPromotion: {
            name: 'prod.01abc.deadbeef',
            status: { phase: 'Running', message: 'working' },
            freight: { name: 'freight-1' }
          }
        })
      )
    ).toEqual({
      kind: 'Promotion',
      name: 'prod.01abc.deadbeef',
      phase: 'Running',
      message: 'working',
      finishedAt: undefined,
      freightName: 'freight-1',
      path: '/project/my-project/promotion/prod.01abc.deadbeef'
    });
  });

  test('target-aware Stage: undefined when no request is fanning out', () => {
    expect(getCurrentPromotionRef(targetAware({}))).toBeUndefined();
  });

  test('target-aware Stage: the current PromotionRequest, linking to the Stage page', () => {
    expect(
      getCurrentPromotionRef(
        targetAware({
          currentPromotionRequest: {
            name: 'prod.01abc.deadbeef',
            phase: 'Running',
            freight: { name: 'freight-1' }
          }
        })
      )
    ).toEqual({
      kind: 'PromotionRequest',
      name: 'prod.01abc.deadbeef',
      phase: 'Running',
      finishedAt: undefined,
      freightName: 'freight-1',
      path: '/project/my-project/stage/prod'
    });
  });

  test('target-aware Stage: ignores the Promotion fields, which name one child at most', () => {
    const ref = getCurrentPromotionRef(
      targetAware({
        currentPromotion: { name: 'prod.target-a.01abc.deadbeef' }
      })
    );
    expect(ref).toBeUndefined();
  });

  test('target-aware Stage: the rollout in flight, linking to its own page', () => {
    const unfinished: PromotionRollout = {
      metadata: { name: 'prod.01abc.deadbeef' },
      number: 7,
      spec: { stage: 'prod', freight: 'freight-1' },
      status: { phase: 'Running', message: '1 of 3 Promotions finished' }
    };
    expect(getCurrentPromotionRef(targetAware({}), unfinished)).toEqual({
      kind: 'PromotionRollout',
      name: 'prod.01abc.deadbeef',
      phase: 'Running',
      message: '1 of 3 Promotions finished',
      finishedAt: undefined,
      freightName: 'freight-1',
      path: '/project/my-project/stage/prod/promotion-rollout/7'
    });
  });

  test('target-aware Stage: the rollout in flight comes before a PromotionRequest', () => {
    const ref = getCurrentPromotionRef(
      targetAware({ currentPromotionRequest: { name: 'prod.request', phase: 'Running' } }),
      { metadata: { name: 'prod.rollout' }, number: 2, status: { phase: 'Running' } }
    );
    expect(ref?.kind).toBe('PromotionRollout');
    expect(ref?.name).toBe('prod.rollout');
  });

  test('target-aware Stage: promoting through a rollout that has not been read yet', () => {
    expect(
      getCurrentPromotionRef(
        targetAware({
          conditions: promoting,
          lastPromotionRollout: { name: 'prod.01abc.deadbeef', number: 6 }
        })
      )
    ).toEqual({
      kind: 'PromotionRollout',
      name: undefined,
      phase: undefined,
      message: undefined,
      finishedAt: undefined,
      freightName: undefined,
      path: '/project/my-project/stage/prod'
    });
  });

  test('target-aware Stage: a recorded rollout alone is not a rollout in flight', () => {
    expect(
      getCurrentPromotionRef(
        targetAware({ lastPromotionRollout: { name: 'prod.01abc.deadbeef', number: 6 } })
      )
    ).toBeUndefined();
  });

  test('target-aware Stage: promoting says nothing of rollouts where none was recorded', () => {
    expect(getCurrentPromotionRef(targetAware({ conditions: promoting }))).toBeUndefined();
  });

  test('classic Stage: has no rollouts, whatever is passed', () => {
    expect(
      getCurrentPromotionRef(classic({}), { metadata: { name: 'x' }, number: 1 })
    ).toBeUndefined();
  });
});

describe('getLastPromotionRef()', () => {
  test('classic Stage: the last Promotion', () => {
    expect(
      getLastPromotionRef(
        classic({
          lastPromotion: {
            name: 'prod.01abc.deadbeef',
            finishedAt: '2026-09-01T10:00:00Z',
            status: { phase: 'Succeeded' },
            freight: { name: 'freight-1', origin: { kind: 'Warehouse', name: 'alpha' } }
          }
        })
      )
    ).toMatchObject({
      kind: 'Promotion',
      name: 'prod.01abc.deadbeef',
      phase: 'Succeeded',
      finishedAt: '2026-09-01T10:00:00Z',
      freightName: 'freight-1',
      freightOrigin: 'alpha',
      path: '/project/my-project/promotion/prod.01abc.deadbeef'
    });
  });

  test('target-aware Stage: the last PromotionRequest', () => {
    expect(
      getLastPromotionRef(
        targetAware({
          lastPromotion: { name: 'prod.target-a.01abc.deadbeef' },
          lastPromotionRequest: {
            name: 'prod.01abc.deadbeef',
            finishedAt: '2026-09-01T10:00:00Z',
            phase: 'Failed',
            freight: { name: 'freight-1', origin: { kind: 'Warehouse', name: 'beta' } }
          }
        })
      )
    ).toMatchObject({
      kind: 'PromotionRequest',
      name: 'prod.01abc.deadbeef',
      phase: 'Failed',
      finishedAt: '2026-09-01T10:00:00Z',
      freightName: 'freight-1',
      freightOrigin: 'beta',
      path: '/project/my-project/stage/prod'
    });
  });

  test('target-aware Stage: the last PromotionRollout, linking to its own page', () => {
    expect(
      getLastPromotionRef(
        targetAware({
          lastPromotionRollout: {
            name: 'prod.01abc.deadbeef',
            number: 12,
            phase: 'Failed',
            message: '1 of 3 Promotions failed',
            finishedAt: '2026-11-16T10:00:00Z'
          }
        })
      )
    ).toEqual({
      kind: 'PromotionRollout',
      name: 'prod.01abc.deadbeef',
      phase: 'Failed',
      message: '1 of 3 Promotions failed',
      finishedAt: '2026-11-16T10:00:00Z',
      freightName: undefined,
      path: '/project/my-project/stage/prod/promotion-rollout/12'
    });
  });

  // Mirrors Stage.LastPromotionName on the back end.
  test('target-aware Stage: a rollout comes before an older PromotionRequest', () => {
    expect(
      getLastPromotionRef(
        targetAware({
          lastPromotionRequest: { name: 'prod.request', finishedAt: '2026-09-01T10:00:00Z' },
          lastPromotionRollout: { name: 'prod.rollout', number: 1 }
        })
      )
    ).toMatchObject({ kind: 'PromotionRollout', name: 'prod.rollout' });
  });

  test('target-aware Stage: a rollout with no number leads to the Stage', () => {
    expect(
      getLastPromotionRef(targetAware({ lastPromotionRollout: { name: 'prod.rollout' } }))?.path
    ).toBe('/project/my-project/stage/prod');
  });

  test('no path when the Stage cannot be located', () => {
    expect(
      getLastPromotionRef(
        stage({
          metadata: {},
          spec: { targets: {} },
          status: { lastPromotionRequest: { name: 'x' } }
        })
      )?.path
    ).toBeUndefined();
    expect(
      getLastPromotionRef(
        stage({ status: { lastPromotion: { finishedAt: '2026-09-01T10:00:00Z' } } })
      )?.path
    ).toBeUndefined();
  });
});

describe('getLastPromotionDate()', () => {
  test('null when nothing has finished', () => {
    expect(getLastPromotionDate(classic({}))).toBeNull();
    expect(getLastPromotionDate(targetAware({}))).toBeNull();
    expect(getLastPromotionDate(classic({ lastPromotion: { name: 'x' } }))).toBeNull();
  });

  test('classic Stage: when the last Promotion finished', () => {
    expect(
      getLastPromotionDate(
        classic({ lastPromotion: { name: 'x', finishedAt: '2026-09-01T10:00:00Z' } })
      )
    ).toEqual(new Date('2026-09-01T10:00:00Z'));
  });

  test('target-aware Stage: when the last PromotionRequest finished', () => {
    expect(
      getLastPromotionDate(
        targetAware({
          lastPromotion: { name: 'x', finishedAt: '2026-08-01T10:00:00Z' },
          lastPromotionRequest: { name: 'y', finishedAt: '2026-09-01T10:00:00Z' }
        })
      )
    ).toEqual(new Date('2026-09-01T10:00:00Z'));
  });
});

describe('getLastPromotionDate(): rollouts', () => {
  test('target-aware Stage: when the last PromotionRollout finished', () => {
    expect(
      getLastPromotionDate(
        targetAware({
          lastPromotionRequest: { name: 'y', finishedAt: '2026-09-01T10:00:00Z' },
          lastPromotionRollout: { name: 'z', number: 3, finishedAt: '2026-11-16T10:00:00Z' }
        })
      )
    ).toEqual(new Date('2026-11-16T10:00:00Z'));
  });
});

describe('getCurrentPromotion()', () => {
  test('classic Stage: the current Promotion name', () => {
    expect(getCurrentPromotion(classic({ currentPromotion: { name: 'x' } }))).toBe('x');
  });

  test('target-aware Stage: never a single Promotion', () => {
    expect(
      getCurrentPromotion(
        targetAware({
          currentPromotion: { name: 'x' },
          currentPromotionRequest: { name: 'y' }
        })
      )
    ).toBeUndefined();
  });
});
