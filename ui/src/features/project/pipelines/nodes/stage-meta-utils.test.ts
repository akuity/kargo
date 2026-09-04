import { describe, expect, test } from 'vitest';

import { Stage } from '@ui/gen/api/v2/models';

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

// targetAware promotes through PromotionRequests: a targets block is present,
// whether or not it currently matches anything.
const targetAware = (status: Stage['status']) => stage({ spec: { targets: {} }, status });

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
});

describe('getLastPromotionRef()', () => {
  test('classic Stage: the last Promotion', () => {
    expect(
      getLastPromotionRef(
        classic({
          lastPromotion: {
            name: 'prod.01abc.deadbeef',
            finishedAt: '2026-09-01T10:00:00Z',
            status: { phase: 'Succeeded' }
          }
        })
      )
    ).toMatchObject({
      kind: 'Promotion',
      name: 'prod.01abc.deadbeef',
      phase: 'Succeeded',
      finishedAt: '2026-09-01T10:00:00Z',
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
            freight: { name: 'freight-1' }
          }
        })
      )
    ).toMatchObject({
      kind: 'PromotionRequest',
      name: 'prod.01abc.deadbeef',
      phase: 'Failed',
      finishedAt: '2026-09-01T10:00:00Z',
      freightName: 'freight-1',
      path: '/project/my-project/stage/prod'
    });
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
