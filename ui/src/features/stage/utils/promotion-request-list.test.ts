import { describe, expect, test } from 'vitest';

import { Freight, PromotionRequest } from '@ui/gen/api/v2/models';

import {
  describeTargets,
  freightVersion,
  phaseFilters,
  requestDurationMs,
  requestPhase,
  roundSummary,
  targetsTally
} from './promotion-request-list';

const request = (partial: Partial<PromotionRequest>): PromotionRequest =>
  ({
    spec: { stage: 'prod-us', freight: 'abc', targets: [] },
    ...partial
  }) as PromotionRequest;

describe('requestPhase()', () => {
  test('reads the recorded phase', () => {
    expect(requestPhase(request({ status: { phase: 'Running' } }))).toBe('Running');
  });

  test('a request the reconciler has not acted on is Pending', () => {
    expect(requestPhase(request({}))).toBe('Pending');
    expect(requestPhase(request({ status: {} }))).toBe('Pending');
    expect(requestPhase(undefined)).toBe('Pending');
  });

  test('a phase this UI does not know reads as Pending', () => {
    expect(requestPhase(request({ status: { phase: 'Rethinking' } }))).toBe('Pending');
  });
});

describe('roundSummary()', () => {
  test('uses the recorded summary', () => {
    const summary = { succeeded: 130, failed: 6 };
    expect(roundSummary(request({ status: { summary } }))).toEqual(summary);
  });

  test('a request that has not fanned out yet has every Target pending', () => {
    expect(
      roundSummary(
        request({
          spec: { stage: 'prod-us', freight: 'abc', targets: [{ name: 'a' }, { name: 'b' }] }
        })
      )
    ).toEqual({ pending: 2 });
  });

  test('a request naming no Targets has nothing pending', () => {
    expect(roundSummary(request({}))).toEqual({ pending: 0 });
  });
});

describe('targetsTally()', () => {
  test('the mockup round: succeeded leads, failed follows', () => {
    const tally = targetsTally({ succeeded: 130, failed: 6 });
    expect(tally.total).toBe(136);
    expect(describeTargets(tally)).toBe('130 succeeded · 6 failed / 136');
  });

  test('a round that entirely succeeded says only that', () => {
    expect(describeTargets(targetsTally({ succeeded: 150 }))).toBe('150 succeeded / 150');
  });

  test('a round in flight shows how far it has got', () => {
    expect(describeTargets(targetsTally({ succeeded: 40, running: 20, pending: 76 }))).toBe(
      '40 succeeded · 20 running · 76 pending / 136'
    );
  });

  test('succeeded is kept at zero so every row reads alike', () => {
    expect(describeTargets(targetsTally({ failed: 3 }))).toBe('0 succeeded · 3 failed / 3');
  });

  test('a round with no children has nothing to tally', () => {
    expect(targetsTally({})).toEqual({ total: 0, parts: [] });
    expect(describeTargets(targetsTally({}))).toBe('no Targets');
  });
});

describe('requestDurationMs()', () => {
  const started = '2026-10-13T03:14:40Z';

  test('a finished round is measured end to end', () => {
    expect(
      requestDurationMs(
        request({ status: { startedAt: started, finishedAt: '2026-10-13T03:18:30Z' } })
      )
    ).toBe(230_000);
  });

  test('a running round is measured against now', () => {
    const now = new Date('2026-10-13T03:16:40Z').getTime();
    expect(requestDurationMs(request({ status: { startedAt: started } }), now)).toBe(120_000);
  });

  test('a round that has not started has no duration', () => {
    expect(requestDurationMs(request({ status: {} }))).toBeUndefined();
    expect(requestDurationMs(undefined)).toBeUndefined();
  });

  test('clock skew yields no duration rather than a negative one', () => {
    const now = new Date('2026-10-13T03:14:00Z').getTime();
    expect(requestDurationMs(request({ status: { startedAt: started } }), now)).toBeUndefined();
  });
});

describe('phaseFilters()', () => {
  const withPhase = (phase: string) => request({ status: { phase } });

  test('the mockup tab: All, then Running, Succeeded and Failed with their counts', () => {
    expect(phaseFilters([withPhase('Failed'), withPhase('Succeeded')])).toEqual([
      { count: 2 },
      { phase: 'Running', count: 0 },
      { phase: 'Succeeded', count: 1 },
      { phase: 'Failed', count: 1 }
    ]);
  });

  test('Running, Succeeded and Failed are offered even when nothing is in them', () => {
    expect(phaseFilters([]).map((filter) => filter.phase)).toEqual([
      undefined,
      'Running',
      'Succeeded',
      'Failed'
    ]);
  });

  test('Pending, Errored and Aborted appear only once they occur', () => {
    const filters = phaseFilters([withPhase('Aborted'), withPhase('Errored'), request({})]);
    expect(filters).toEqual([
      { count: 3 },
      { phase: 'Running', count: 0 },
      { phase: 'Pending', count: 1 },
      { phase: 'Succeeded', count: 0 },
      { phase: 'Failed', count: 0 },
      { phase: 'Errored', count: 1 },
      { phase: 'Aborted', count: 1 }
    ]);
  });
});

describe('freightVersion()', () => {
  const freight = (partial: Partial<Freight>): Freight => partial as Freight;

  test('names the version of a single image', () => {
    expect(freightVersion(freight({ images: [{ tag: '0.2.0' }] }))).toBe('0.2.0');
  });

  test('names a chart or artifact version too', () => {
    expect(freightVersion(freight({ charts: [{ version: '1.4.2' }] }))).toBe('1.4.2');
    expect(freightVersion(freight({ artifacts: [{ version: 'v3' }] }))).toBe('v3');
  });

  test('artifacts that agree on a version name it once', () => {
    expect(freightVersion(freight({ images: [{ tag: '0.2.0' }, { tag: '0.2.0' }] }))).toBe('0.2.0');
  });

  test('artifacts that disagree are ambiguous and go unnamed', () => {
    expect(
      freightVersion(freight({ images: [{ tag: '0.2.0' }, { tag: '0.3.0' }] }))
    ).toBeUndefined();
  });

  test('a commit is not a version', () => {
    expect(freightVersion(freight({ commits: [{ id: 'abc1234', tag: 'v1' }] }))).toBeUndefined();
  });

  test('Freight carrying nothing versioned has no version', () => {
    expect(freightVersion(freight({}))).toBeUndefined();
    expect(freightVersion(undefined)).toBeUndefined();
  });
});
