import { describe, expect, test } from 'vitest';

import { PromotionRequest } from '@ui/gen/api/v2/models';

import {
  blockingMessage,
  isPromotionRequestPhaseTerminal,
  promotionRequestCompareFn,
  targetRows
} from './promotion-request';

const request = (partial: Partial<PromotionRequest>): PromotionRequest =>
  ({
    spec: { stage: 'prod', freight: 'abc', targets: [] },
    ...partial
  }) as PromotionRequest;

describe('isPromotionRequestPhaseTerminal()', () => {
  test.each([
    ['Succeeded', true],
    ['Failed', true],
    ['Errored', true],
    ['Pending', false],
    ['Running', false],
    [undefined, false]
  ])('%s -> %s', (phase, terminal) => {
    expect(isPromotionRequestPhaseTerminal(phase)).toBe(terminal);
  });
});

describe('promotionRequestCompareFn()', () => {
  test('orders newest (greatest name) first', () => {
    // Names embed a ULID, so lexical order over names is creation order.
    const older = request({ metadata: { name: 'prod.01aaa.1111' } });
    const newer = request({ metadata: { name: 'prod.01bbb.2222' } });
    expect([older, newer].sort(promotionRequestCompareFn)).toEqual([newer, older]);
  });

  test('tolerates a missing name', () => {
    const named = request({ metadata: { name: 'prod.01aaa.1111' } });
    const nameless = request({});
    expect([nameless, named].sort(promotionRequestCompareFn)).toEqual([named, nameless]);
  });
});

describe('blockingMessage()', () => {
  test('returns the Ready=False message', () => {
    expect(
      blockingMessage(
        request({
          status: {
            conditions: [
              {
                type: 'Ready',
                status: 'False',
                message: 'PromotionRequests are a Kargo Enterprise-only feature',
                lastTransitionTime: '2026-08-13T00:00:00Z',
                reason: 'EnterpriseOnlyFeature'
              }
            ]
          }
        })
      )
    ).toBe('PromotionRequests are a Kargo Enterprise-only feature');
  });

  test('returns nothing when Ready is not False', () => {
    expect(
      blockingMessage(
        request({
          status: {
            conditions: [
              {
                type: 'Ready',
                status: 'True',
                message: 'all good',
                lastTransitionTime: '2026-08-13T00:00:00Z',
                reason: 'Ready'
              }
            ]
          }
        })
      )
    ).toBeUndefined();
  });

  test('returns nothing when there is no Ready condition', () => {
    expect(blockingMessage(request({}))).toBeUndefined();
    expect(blockingMessage(undefined)).toBeUndefined();
  });
});

describe('targetRows()', () => {
  test('returns a bare row per named Target with no status yet', () => {
    expect(
      targetRows(
        request({
          spec: { stage: 'prod', freight: 'abc', targets: [{ name: 'east' }, { name: 'west' }] }
        })
      )
    ).toEqual([{ name: 'east' }, { name: 'west' }]);
  });

  test('merges status entries onto the named Targets', () => {
    expect(
      targetRows(
        request({
          spec: { stage: 'prod', freight: 'abc', targets: [{ name: 'east' }, { name: 'west' }] },
          status: {
            targets: [{ name: 'east', promotion: 'prod.east.01aaa.1111', phase: 'Running' }]
          }
        })
      )
    ).toEqual([
      { name: 'east', promotion: 'prod.east.01aaa.1111', phase: 'Running' },
      { name: 'west' }
    ]);
  });

  // The governing Stage may add Targets to an in-flight request; the status
  // can therefore reference a Target the (stale, cached) spec does not name.
  test('includes status entries for Targets the spec does not name', () => {
    expect(
      targetRows(
        request({
          spec: { stage: 'prod', freight: 'abc', targets: [{ name: 'east' }] },
          status: {
            targets: [{ name: 'south', promotion: 'prod.south.01aaa.1111', phase: 'Pending' }]
          }
        })
      )
    ).toEqual([
      { name: 'east' },
      { name: 'south', promotion: 'prod.south.01aaa.1111', phase: 'Pending' }
    ]);
  });

  test('returns no rows for a request naming no Targets', () => {
    expect(targetRows(request({}))).toEqual([]);
    expect(targetRows(undefined)).toEqual([]);
  });
});
