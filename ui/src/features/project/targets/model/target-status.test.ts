import { describe, expect, test } from 'vitest';

import { emptyStateCounts } from './target-state';
import {
  fleetFigures,
  shouldPollList,
  shouldPollStatuses,
  shouldPollSummary,
  stageFigures,
  statusSignature,
  targetsSignature
} from './target-status';
import { entry, fleetStage, target } from './test-fixtures';

const summary = {
  targets: 12,
  behind: 4,
  states: {
    Promoting: 1,
    Pending: 2,
    Failed: 1,
    NotAttempted: 3,
    NeverPromoted: 0,
    Behind: 1,
    UpToDate: 2
  },
  stages: [
    {
      stage: 'prod',
      behind: 3,
      states: { ...emptyStateCounts(), Failed: 1, NotAttempted: 3, UpToDate: 4 }
    }
  ]
};

describe('fleetFigures()', () => {
  test("reads the server's counts, every state present", () => {
    const figures = fleetFigures(summary);
    expect(figures.counts.NotAttempted).toBe(3);
    expect(figures.behind).toBe(4);
  });

  test('counts the Targets no Stage governs as never promoted, as the rows do', () => {
    // 10 governed, 12 in the Project.
    const figures = fleetFigures(summary);
    expect(figures.counts.NeverPromoted).toBe(2);
    expect(figures.total).toBe(12);
  });

  test('no summary is an empty fleet', () => {
    expect(fleetFigures(undefined)).toEqual({ counts: emptyStateCounts(), behind: 0, total: 0 });
  });
});

describe('stageFigures()', () => {
  test("reads one Stage's counts", () => {
    const figures = stageFigures(summary, 'prod');
    expect(figures.counts.UpToDate).toBe(4);
    expect(figures.behind).toBe(3);
    expect(figures.total).toBe(8);
  });

  test('a Stage the summary does not hold has nothing counted', () => {
    expect(stageFigures(summary, 'dev').total).toBe(0);
  });
});

describe('shouldPollStatuses()', () => {
  const idle = fleetStage('prod', [{}]);
  const promoting = {
    ...idle,
    status: {
      conditions: [
        {
          type: 'Promoting',
          status: 'True',
          reason: 'ActivePromotion',
          message: '',
          lastTransitionTime: ''
        }
      ]
    }
  };

  test('polls while a promotion is on its way to any Target', () => {
    expect(shouldPollStatuses([entry('a', 'prod', 'Pending')], [idle])).toBe(true);
    expect(shouldPollStatuses([entry('a', 'prod', 'Promoting')], [idle])).toBe(true);
  });

  test('polls while a Stage says it is promoting, before the statuses show it', () => {
    expect(shouldPollStatuses([entry('a', 'prod', 'UpToDate')], [promoting])).toBe(true);
  });

  test('stops when nothing is in flight', () => {
    expect(
      shouldPollStatuses([entry('a', 'prod', 'UpToDate'), entry('b', 'prod', 'Failed')], [idle])
    ).toBe(false);
    expect(shouldPollStatuses([], [])).toBe(false);
  });
});

describe('shouldPollList()', () => {
  const idle = fleetStage('prod', [{}]);
  const settled = [entry('a', 'prod', 'UpToDate')];

  test('reads again after a failed read, though nothing is in flight', () => {
    // Nothing else retries a failed read, and a first read that fails has no
    // entries to say anything is in flight.
    expect(shouldPollList({ failed: true, updatedAt: 0 }, [], [idle])).toBe(true);
  });

  test('otherwise goes by whether anything is in flight', () => {
    expect(shouldPollList({ failed: false, updatedAt: 1 }, settled, [idle])).toBe(false);
    expect(
      shouldPollList({ failed: false, updatedAt: 1 }, [entry('a', 'prod', 'Pending')], [idle])
    ).toBe(true);
  });

  test('stops after a successful read that found nothing', () => {
    expect(shouldPollList({ failed: false, updatedAt: 1 }, [], [idle])).toBe(false);
  });
});

describe('shouldPollSummary()', () => {
  const idle = fleetStage('prod', [{}]);
  const settled = [entry('a', 'prod', 'UpToDate')];
  const list = { failed: false, updatedAt: 2000 };

  test('reads once more when the list was read after it', () => {
    // The summary read the mirror before a promotion finished; the list read
    // it after, found nothing in flight, and stopped polling.
    expect(shouldPollSummary({ failed: false, updatedAt: 1000 }, list, settled, [idle])).toBe(true);
  });

  test('stops once a read of it follows the list', () => {
    expect(shouldPollSummary({ failed: false, updatedAt: 3000 }, list, settled, [idle])).toBe(
      false
    );
  });

  test('reads again after a failed read of it or of the list', () => {
    expect(shouldPollSummary({ failed: true, updatedAt: 3000 }, list, settled, [idle])).toBe(true);
    expect(
      shouldPollSummary(
        { failed: false, updatedAt: 3000 },
        { failed: true, updatedAt: 0 },
        [],
        [idle]
      )
    ).toBe(true);
  });

  test("goes by the list's entries while something is in flight", () => {
    expect(
      shouldPollSummary(
        { failed: false, updatedAt: 3000 },
        list,
        [entry('a', 'prod', 'Promoting')],
        [idle]
      )
    ).toBe(true);
  });
});

describe('statusSignature()', () => {
  const stage = fleetStage('prod', [{}], 'f1');

  test('changes when a Stage records a new round or moves to new Freight', () => {
    const before = statusSignature([stage]);
    expect(
      statusSignature([
        { ...stage, status: { ...stage.status, lastPromotionRollout: { name: 'prod.r2' } } }
      ])
    ).not.toBe(before);
    expect(statusSignature([fleetStage('prod', [{}], 'f2')])).not.toBe(before);
  });

  test('changes when a Stage selects other Targets', () => {
    const before = statusSignature([fleetStage('prod', [{ matchLabels: { env: 'prod' } }], 'f1')]);
    expect(
      statusSignature([fleetStage('prod', [{ matchLabels: { env: 'prod', tier: 'a' } }], 'f1')])
    ).not.toBe(before);
  });

  test('does not depend on the order of the Stages', () => {
    const other = fleetStage('dev', [{}], 'f1');
    expect(statusSignature([stage, other])).toBe(statusSignature([other, stage]));
  });
});

describe('targetsSignature()', () => {
  const a = target('a', { env: 'prod', tier: 'one' });
  const b = target('b', { env: 'prod' });

  test('changes when a Target comes or goes', () => {
    const before = targetsSignature([a, b]);
    expect(targetsSignature([a])).not.toBe(before);
    expect(targetsSignature([a, b, target('c', { env: 'prod' })])).not.toBe(before);
  });

  test('changes when a Target is relabelled', () => {
    expect(targetsSignature([a, target('b', { env: 'dev' })])).not.toBe(targetsSignature([a, b]));
  });

  test('does not depend on the order of the Targets or of their labels', () => {
    expect(targetsSignature([b, a])).toBe(targetsSignature([a, b]));
    expect(targetsSignature([target('a', { tier: 'one', env: 'prod' }), b])).toBe(
      targetsSignature([a, b])
    );
  });
});
