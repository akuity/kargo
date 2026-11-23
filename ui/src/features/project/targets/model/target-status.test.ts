import { describe, expect, test } from 'vitest';

import { emptyStateCounts } from './target-state';
import { fleetFigures, shouldPollStatuses, stageFigures, statusSignature } from './target-status';
import { entry, fleetStage } from './test-fixtures';

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

  test('does not depend on the order of the Stages', () => {
    const other = fleetStage('dev', [{}], 'f1');
    expect(statusSignature([stage, other])).toBe(statusSignature([other, stage]));
  });
});
