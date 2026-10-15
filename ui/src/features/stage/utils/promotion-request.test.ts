import { describe, expect, test } from 'vitest';

import { PromotionRequest, Stage } from '@ui/gen/api/v2/models';

import {
  blockingMessage,
  currentRound,
  currentRounds,
  isPromotionRequestPhaseTerminal,
  promotionRequestCompareFn,
  roundBlock,
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
    // Mirrors PromotionRequestPhase.IsTerminal on the back end.
    ['Aborted', true],
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
  test('prefers status.message', () => {
    expect(
      blockingMessage(
        request({
          status: {
            message: 'Waiting on 2 of 3 Promotions to complete',
            conditions: [
              {
                type: 'Ready',
                status: 'False',
                message: 'operator-facing detail',
                lastTransitionTime: '2026-08-13T00:00:00Z',
                reason: 'Promoting'
              }
            ]
          }
        })
      )
    ).toBe('Waiting on 2 of 3 Promotions to complete');
  });

  test('falls back to the Ready=False message', () => {
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

describe('roundBlock()', () => {
  const blocked = (reason: string, message: string) =>
    request({
      status: {
        conditions: [
          {
            type: 'Ready',
            status: 'False',
            reason,
            message,
            lastTransitionTime: '2026-08-13T00:00:00Z'
          }
        ]
      }
    });

  test('explains the Enterprise-only reason in plain language', () => {
    const block = roundBlock(
      blocked('EnterpriseOnlyFeature', 'PromotionRequests are a Kargo Enterprise-only feature')
    );
    expect(block?.title).toBe('Fleet management is a Kargo Enterprise feature');
    expect(block?.description).toContain('does not support');
  });

  test('passes any other reason through under a general heading', () => {
    const block = roundBlock(blocked('TargetsMissing', 'Target us-east-1 not found'));
    expect(block?.title).toBe('Promotion to this Stage cannot progress');
    expect(block?.description).toBe('Target us-east-1 not found');
  });

  test('is not a block once the round has fanned out, even with failed children', () => {
    const failedRound = blocked('PromotionsFailed', '1 of 3 Promotions did not succeed');
    failedRound.status = {
      ...failedRound.status,
      phase: 'Errored',
      summary: { succeeded: 2, errored: 1 },
      targets: [{ name: 'eu', promotion: 'p', phase: 'Errored' }]
    };
    expect(roundBlock(failedRound)).toBeUndefined();
  });

  test('is undefined when the request is not blocked', () => {
    expect(roundBlock(undefined)).toBeUndefined();
    expect(roundBlock(request({}))).toBeUndefined();
    expect(
      roundBlock(
        request({
          status: {
            conditions: [
              { type: 'Ready', status: 'True', lastTransitionTime: '2026-08-13T00:00:00Z' }
            ]
          }
        })
      )
    ).toBeUndefined();
  });
});

describe('currentRound()', () => {
  const named = (name: string, stage = 'prod') =>
    request({ metadata: { name }, spec: { stage, freight: 'abc', targets: [] } });
  const prod = (status?: Stage['status']): Stage => ({ metadata: { name: 'prod' }, status });
  const requests = [named('prod.01'), named('prod.03'), named('prod.02'), named('other.09', 'uat')];

  test("prefers the round the Stage's status calls current", () => {
    const stage = prod({
      currentPromotionRequest: { name: 'prod.01' },
      lastPromotionRequest: { name: 'prod.02' }
    } as Stage['status']);
    expect(currentRound(stage, requests)?.metadata?.name).toBe('prod.01');
  });

  test('falls back to the last round the status names', () => {
    const stage = prod({ lastPromotionRequest: { name: 'prod.02' } } as Stage['status']);
    expect(currentRound(stage, requests)?.metadata?.name).toBe('prod.02');
  });

  test('falls back to the newest round the Stage has, ignoring other Stages', () => {
    expect(currentRound(prod(), requests)?.metadata?.name).toBe('prod.03');
  });

  test('a status naming a round of another Stage does not reach it', () => {
    const stage = prod({ currentPromotionRequest: { name: 'other.09' } } as Stage['status']);
    expect(currentRound(stage, requests)?.metadata?.name).toBe('prod.03');
  });

  test('a Stage with no rounds, or no name, has none', () => {
    expect(currentRound(prod(), [])).toBeUndefined();
    expect(currentRound(undefined, requests)).toBeUndefined();
  });
});

describe('currentRounds()', () => {
  const named = (name: string, stage: string) =>
    request({ metadata: { name }, spec: { stage, freight: 'abc', targets: [] } });
  const requests = [
    named('prod.01', 'prod'),
    named('prod.03', 'prod'),
    named('uat.09', 'uat'),
    named('prod.02', 'prod')
  ];

  test('resolves every Stage against its own requests', () => {
    const rounds = currentRounds(
      [
        { metadata: { name: 'prod' } },
        { metadata: { name: 'uat' }, status: { lastPromotionRequest: { name: 'uat.09' } } },
        { metadata: { name: 'never-promoted' } }
      ] as Stage[],
      requests
    );
    expect(rounds.prod?.metadata?.name).toBe('prod.03');
    expect(rounds.uat?.metadata?.name).toBe('uat.09');
    expect(rounds['never-promoted']).toBeUndefined();
  });

  test('agrees with currentRound, Stage for Stage', () => {
    const stages = [{ metadata: { name: 'prod' } }, { metadata: { name: 'uat' } }] as Stage[];
    const rounds = currentRounds(stages, requests);
    for (const stage of stages) {
      expect(rounds[stage.metadata?.name || '']).toBe(currentRound(stage, requests));
    }
  });

  test('a Stage with no name is left out', () => {
    expect(currentRounds([{} as Stage], requests)).toEqual({});
  });
});
