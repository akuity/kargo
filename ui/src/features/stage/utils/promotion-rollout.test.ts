import { describe, expect, test } from 'vitest';

import { PromotionRequest, PromotionRollout, Stage } from '@ui/gen/api/v2/models';

import {
  canAbortRollout,
  currentRollout,
  isRolloutAbortRequested,
  isRolloutAborting,
  isRolloutFinished,
  isRolloutRound,
  isStagePromoting,
  latestRolloutsByStage,
  promotedRoundPath,
  recordedRollouts,
  rolloutCompareFn,
  rolloutPath,
  rolloutRound,
  rolloutRounds,
  rolloutToRound,
  roundIdentity,
  roundMessage,
  roundPath,
  shouldPollRollouts,
  stageUsesRollouts
} from './promotion-rollout';

const rollout = (
  number: number,
  phase?: string,
  partial: Partial<PromotionRollout> = {}
): PromotionRollout =>
  ({
    metadata: { name: `prod.rollout-${number}` },
    number,
    spec: { stage: 'prod', freight: 'abc', targets: [] },
    status: phase ? { phase } : undefined,
    ...partial
  }) as PromotionRollout;

const stage = (name: string, status: Stage['status'] = {}): Stage =>
  ({
    metadata: { namespace: 'my-project', name },
    spec: { targets: {} },
    status
  }) as Stage;

const promoting = { conditions: [{ type: 'Promoting', status: 'True' }] } as Stage['status'];

describe('rolloutToRound()', () => {
  test('carries what a round is read by, marked as a rollout', () => {
    const round = rolloutToRound({
      metadata: {
        name: 'prod.01abc.deadbeef',
        creationTimestamp: '2026-11-16T10:00:00Z',
        annotations: { 'kargo.akuity.io/create-actor': 'admin' }
      },
      number: 7,
      spec: {
        stage: 'prod',
        freight: 'abc',
        targets: [{ name: 'east' }, { name: 'west' }],
        updateStrategy: { maxConcurrent: 2, maxFailures: 1 }
      },
      status: {
        phase: 'Running',
        message: '1 of 2 Promotions finished',
        startedAt: '2026-11-16T10:00:01Z',
        conditions: [{ type: 'Ready', status: 'Unknown', reason: 'Promoting' }],
        summary: { succeeded: 1, running: 1 },
        targets: [
          { name: 'east', promotion: 'prod.east.01abc', phase: 'Succeeded' },
          { name: 'west', promotion: 'prod.west.01abc', phase: 'Running' }
        ]
      }
    } as PromotionRollout);

    expect(round).toEqual({
      kind: 'PromotionRollout',
      metadata: {
        name: 'prod.01abc.deadbeef',
        creationTimestamp: '2026-11-16T10:00:00Z',
        annotations: { 'kargo.akuity.io/create-actor': 'admin' }
      },
      number: 7,
      spec: {
        stage: 'prod',
        freight: 'abc',
        targets: [{ name: 'east' }, { name: 'west' }],
        updateStrategy: { maxConcurrent: 2, maxFailures: 1 }
      },
      status: {
        phase: 'Running',
        message: '1 of 2 Promotions finished',
        startedAt: '2026-11-16T10:00:01Z',
        finishedAt: undefined,
        conditions: [{ type: 'Ready', status: 'Unknown', reason: 'Promoting' }],
        summary: { succeeded: 1, running: 1 },
        targets: [
          { name: 'east', promotion: 'prod.east.01abc', phase: 'Succeeded' },
          { name: 'west', promotion: 'prod.west.01abc', phase: 'Running' }
        ]
      }
    });
  });

  test('a rollout with nothing recorded yet is still a round', () => {
    expect(rolloutToRound({ number: 1 })).toEqual({
      kind: 'PromotionRollout',
      metadata: undefined,
      number: 1,
      spec: { stage: '', freight: '', targets: [], updateStrategy: undefined },
      status: undefined
    });
  });

  test('drops Targets that carry no name, which nothing could be said about', () => {
    const round = rolloutToRound({
      number: 1,
      spec: { targets: [{ name: 'east' }, {}] },
      status: { targets: [{ phase: 'Running' }, { name: 'east', phase: 'Succeeded' }] }
    });
    expect(round.spec.targets).toEqual([{ name: 'east' }]);
    expect(round.status?.targets).toEqual([{ name: 'east', phase: 'Succeeded' }]);
  });

  // A rollout may gain fields; a round shows only the ones a view reads.
  test('leaves out fields a round is not read by', () => {
    const round = rolloutToRound({
      number: 1,
      status: { phase: 'Running', somethingNew: 'later' }
    } as PromotionRollout);
    expect(round.status).not.toHaveProperty('somethingNew');
  });
});

describe('isRolloutRound()', () => {
  test('a rollout presented as a round', () => {
    expect(isRolloutRound(rolloutToRound(rollout(3)))).toBe(true);
  });

  test('a PromotionRequest is not one', () => {
    const request = {
      kind: 'PromotionRequest',
      spec: { stage: 'prod', freight: 'abc', targets: [] }
    } as PromotionRequest;
    expect(isRolloutRound(request)).toBe(false);
    expect(isRolloutRound(undefined)).toBe(false);
  });

  test('nor is a rollout with no number to read it back by', () => {
    expect(isRolloutRound(rolloutToRound({ spec: { stage: 'prod' } }))).toBe(false);
  });
});

describe('roundIdentity()', () => {
  test('a rollout is known by its number, with its name as a hint', () => {
    expect(roundIdentity(rolloutToRound(rollout(12)))).toEqual({
      label: '#12',
      hint: 'prod.rollout-12'
    });
  });

  test('a PromotionRequest is known by its name', () => {
    const request = {
      metadata: { name: 'prod.01abc.deadbeef' },
      spec: { stage: 'prod', freight: 'abc', targets: [] }
    } as PromotionRequest;
    expect(roundIdentity(request)).toEqual({ label: 'prod.01abc.deadbeef' });
  });

  test('no round, nothing to call it', () => {
    expect(roundIdentity(undefined)).toEqual({ label: '' });
  });
});

describe('roundMessage()', () => {
  const withMessage = (phase: string, message?: string) =>
    rolloutToRound(rollout(1, phase, { status: { phase, message } } as Partial<PromotionRollout>));

  test('a rollout that succeeded has nothing to add to its tally', () => {
    expect(roundMessage(withMessage('Succeeded', 'All 3 Promotions succeeded'))).toBeUndefined();
  });

  test.each(['Pending', 'Running', 'Failed', 'Errored', 'Aborted'])(
    'a rollout that is %s says why',
    (phase) => {
      expect(roundMessage(withMessage(phase, 'something to say'))).toBe('something to say');
    }
  );

  test('a PromotionRequest keeps its message whatever its phase', () => {
    const request = {
      metadata: { name: 'prod.01abc.deadbeef' },
      spec: { stage: 'prod', freight: 'abc', targets: [] },
      status: { phase: 'Succeeded', message: 'All 3 Promotions succeeded' }
    } as PromotionRequest;
    expect(roundMessage(request)).toBe('All 3 Promotions succeeded');
  });

  test('no message, nothing to show', () => {
    expect(roundMessage(withMessage('Failed'))).toBeUndefined();
    expect(roundMessage(withMessage('Failed', ''))).toBeUndefined();
    expect(roundMessage(undefined)).toBeUndefined();
  });
});

describe('roundPath()', () => {
  test('a rollout is read by its number', () => {
    expect(rolloutPath('my-project', 'prod', 12)).toBe(
      '/project/my-project/stage/prod/promotion-rollout/12'
    );
    expect(roundPath('my-project', 'prod', rolloutToRound(rollout(12)))).toBe(
      '/project/my-project/stage/prod/promotion-rollout/12'
    );
  });

  test('a PromotionRequest is read by its name', () => {
    const request = {
      metadata: { name: 'prod.01abc.deadbeef' },
      spec: { stage: 'prod', freight: 'abc', targets: [] }
    } as PromotionRequest;
    expect(roundPath('my-project', 'prod', request)).toBe(
      '/project/my-project/stage/prod/promotion-request/prod.01abc.deadbeef'
    );
  });

  test('a round with neither has nowhere to lead', () => {
    expect(roundPath('my-project', 'prod', undefined)).toBeUndefined();
    expect(
      roundPath('my-project', 'prod', {
        spec: { stage: 'prod', freight: 'abc', targets: [] }
      })
    ).toBeUndefined();
  });
});

describe('promotedRoundPath()', () => {
  test('the server created a PromotionRollout', () => {
    expect(
      promotedRoundPath('my-project', 'prod', {
        metadata: { name: 'prod.01abc.deadbeef' },
        number: 5
      })
    ).toBe('/project/my-project/stage/prod/promotion-rollout/5');
  });

  test('the server created a PromotionRequest', () => {
    expect(
      promotedRoundPath('my-project', 'prod', { metadata: { name: 'prod.01abc.deadbeef' } })
    ).toBe('/project/my-project/stage/prod/promotion-request/prod.01abc.deadbeef');
  });

  test('nothing to land on', () => {
    expect(promotedRoundPath('my-project', 'prod', undefined)).toBeUndefined();
    expect(promotedRoundPath('my-project', 'prod', {})).toBeUndefined();
  });
});

describe('aborting a rollout', () => {
  const abortRequested = {
    metadata: {
      name: 'prod.rollout-1',
      annotations: { 'kargo.akuity.io/abort': '{"action":"terminate","actor":"alice"}' }
    }
  };

  test.each([
    ['Pending', true],
    ['Running', true],
    [undefined, true],
    ['Succeeded', false],
    ['Failed', false],
    ['Errored', false],
    ['Aborted', false]
  ])('a %s rollout can be aborted: %s', (phase, abortable) => {
    expect(canAbortRollout(rolloutToRound(rollout(1, phase)))).toBe(abortable);
  });

  test('a rollout asked to abort cannot be asked again, and is aborting until it ends', () => {
    const running = rolloutToRound(rollout(1, 'Running', abortRequested));
    expect(canAbortRollout(running)).toBe(false);
    expect(isRolloutAbortRequested(running)).toBe(true);
    expect(isRolloutAborting(running)).toBe(true);

    const aborted = rolloutToRound(rollout(1, 'Aborted', abortRequested));
    expect(isRolloutAbortRequested(aborted)).toBe(true);
    expect(isRolloutAborting(aborted)).toBe(false);
  });

  test('a rollout no one asked to abort is not aborting', () => {
    const running = rolloutToRound(rollout(1, 'Running'));
    expect(isRolloutAbortRequested(running)).toBe(false);
    expect(isRolloutAborting(running)).toBe(false);
  });

  test('a PromotionRequest is not aborted from here', () => {
    const request = { metadata: { name: 'req' }, status: { phase: 'Running' } } as PromotionRequest;
    expect(canAbortRollout(request)).toBe(false);
    expect(isRolloutAborting({ ...request, ...abortRequested })).toBe(false);
  });
});

describe('isRolloutFinished()', () => {
  test.each([
    ['Succeeded', true],
    ['Failed', true],
    ['Errored', true],
    ['Aborted', true],
    ['Pending', false],
    ['Running', false],
    [undefined, false]
  ])('%s -> %s', (phase, finished) => {
    expect(isRolloutFinished(rollout(1, phase))).toBe(finished);
  });

  test('no rollout at all is not finished', () => {
    expect(isRolloutFinished(undefined)).toBe(false);
  });
});

describe('rolloutCompareFn()', () => {
  test('orders newest first, by number', () => {
    const sorted = [rollout(2), rollout(10), rollout(1)].sort(rolloutCompareFn);
    expect(sorted.map((r) => r.number)).toEqual([10, 2, 1]);
  });
});

describe('latestRolloutsByStage()', () => {
  test('no rollouts, no Stages', () => {
    expect(latestRolloutsByStage([])).toEqual({});
  });

  test('keeps the rollout in flight and the newest that finished', () => {
    const running = rollout(4, 'Running');
    const succeeded = rollout(3, 'Succeeded');
    expect(latestRolloutsByStage([rollout(2, 'Failed'), succeeded, running])).toEqual({
      prod: { unfinished: running, finished: succeeded }
    });
  });

  test('a Stage with nothing in flight has only a finished rollout', () => {
    const failed = rollout(2, 'Failed');
    expect(latestRolloutsByStage([rollout(1, 'Succeeded'), failed])).toEqual({
      prod: { finished: failed }
    });
  });

  test('a rollout that has recorded no phase is in flight', () => {
    const fresh = rollout(1);
    expect(latestRolloutsByStage([fresh])).toEqual({ prod: { unfinished: fresh } });
  });

  test('separates the rollouts of a whole Project by Stage', () => {
    const prod = rollout(9, 'Succeeded');
    const staging = rollout(2, 'Running', {
      spec: { stage: 'staging', freight: 'abc', targets: [] }
    });
    expect(latestRolloutsByStage([prod, staging])).toEqual({
      prod: { finished: prod },
      staging: { unfinished: staging }
    });
  });

  test('ignores a rollout that names no Stage', () => {
    expect(latestRolloutsByStage([{ number: 1 }])).toEqual({});
  });

  test('a Stage named for something every object inherits is a Stage like any other', () => {
    const finished = rollout(1, 'Succeeded', {
      spec: { stage: 'constructor', freight: 'abc', targets: [] }
    });
    const byStage = latestRolloutsByStage([finished]);
    expect(byStage.constructor).toEqual({ finished });
    expect(latestRolloutsByStage([]).constructor).toBeUndefined();
    expect(Object).not.toHaveProperty('finished');
  });
});

describe('stageUsesRollouts()', () => {
  test('a Stage that has recorded a rollout', () => {
    expect(
      stageUsesRollouts(stage('prod', { lastPromotionRollout: { name: 'prod.rollout-1' } }))
    ).toBe(true);
  });

  test("a Stage with a rollout it has nothing to record of yet: it's first", () => {
    expect(stageUsesRollouts(stage('prod'), { unfinished: rollout(1, 'Running') })).toBe(true);
    expect(stageUsesRollouts(stage('prod'), { finished: rollout(1, 'Succeeded') })).toBe(true);
  });

  test('a Stage with neither keeps its PromotionRequests', () => {
    expect(stageUsesRollouts(stage('prod'))).toBe(false);
    expect(stageUsesRollouts(stage('prod'), {})).toBe(false);
    expect(
      stageUsesRollouts(stage('prod', { lastPromotionRequest: { name: 'prod.01abc.deadbeef' } }))
    ).toBe(false);
    expect(stageUsesRollouts(undefined)).toBe(false);
  });

  test('a rollout wins over the reference of an older PromotionRequest', () => {
    expect(
      stageUsesRollouts(
        stage('prod', {
          lastPromotionRequest: { name: 'prod.01abc.deadbeef' },
          lastPromotionRollout: { name: 'prod.rollout-1' }
        })
      )
    ).toBe(true);
  });
});

describe('currentRollout()', () => {
  test('the rollout in flight comes first', () => {
    const running = rollout(4, 'Running');
    expect(currentRollout({ unfinished: running, finished: rollout(3, 'Succeeded') })).toBe(
      running
    );
  });

  test('else the newest that finished', () => {
    const succeeded = rollout(3, 'Succeeded');
    expect(currentRollout({ finished: succeeded })).toBe(succeeded);
  });

  test('else nothing', () => {
    expect(currentRollout({})).toBeUndefined();
    expect(currentRollout(undefined)).toBeUndefined();
  });
});

describe('rolloutRound()', () => {
  test('the current rollout, as a round', () => {
    expect(rolloutRound({ unfinished: rollout(4, 'Running') })).toMatchObject({
      kind: 'PromotionRollout',
      number: 4,
      status: { phase: 'Running' }
    });
  });

  test('nothing for a Stage with no rollout to show', () => {
    expect(rolloutRound(undefined)).toBeUndefined();
  });
});

describe('rolloutRounds()', () => {
  test('resolves a round for each Stage on rollouts and leaves the others out', () => {
    const rounds = rolloutRounds(
      [
        stage('prod', { lastPromotionRollout: { name: 'prod.rollout-3' } }),
        stage('staging'),
        stage('requests', { lastPromotionRequest: { name: 'requests.01abc.deadbeef' } })
      ],
      {
        prod: { finished: rollout(3, 'Succeeded') },
        staging: {
          unfinished: rollout(1, 'Running', {
            spec: { stage: 'staging', freight: 'abc', targets: [] }
          })
        }
      }
    );
    expect(Object.keys(rounds).sort()).toEqual(['prod', 'staging']);
    expect(rounds.prod).toMatchObject({ number: 3, status: { phase: 'Succeeded' } });
    expect(rounds.staging).toMatchObject({ number: 1, status: { phase: 'Running' } });
  });

  // The entry is there, and empty, so that the Stage's PromotionRequests are
  // not shown in the round's place while the list catches up.
  test('a Stage whose rollout the list does not hold yet has no round for now', () => {
    const rounds = rolloutRounds(
      [stage('prod', { lastPromotionRollout: { name: 'prod.rollout-3' } })],
      {}
    );
    expect(rounds).toEqual({ prod: undefined });
    expect('prod' in rounds).toBe(true);
  });

  test('skips a Stage with no name', () => {
    expect(
      rolloutRounds([{ status: { lastPromotionRollout: { name: 'x' } } } as Stage], {})
    ).toEqual({});
  });
});

describe('isStagePromoting()', () => {
  test('reads the Promoting condition', () => {
    expect(isStagePromoting(stage('prod', promoting))).toBe(true);
    expect(
      isStagePromoting(stage('prod', { conditions: [{ type: 'Promoting', status: 'False' }] }))
    ).toBe(false);
    expect(isStagePromoting(stage('prod'))).toBe(false);
    expect(isStagePromoting(undefined)).toBe(false);
  });
});

describe('shouldPollRollouts()', () => {
  test('nothing in flight: the list is left alone', () => {
    expect(shouldPollRollouts([stage('prod')], [rollout(1, 'Succeeded')])).toBe(false);
    expect(shouldPollRollouts([], [])).toBe(false);
  });

  test('a rollout in the list has yet to finish', () => {
    expect(shouldPollRollouts([stage('prod')], [rollout(2, 'Running')])).toBe(true);
    expect(shouldPollRollouts([stage('prod')], [rollout(2, 'Pending')])).toBe(true);
  });

  test('a Stage says it is promoting, though the list shows nothing in flight', () => {
    expect(shouldPollRollouts([stage('prod', promoting)], [rollout(1, 'Succeeded')])).toBe(true);
    expect(shouldPollRollouts([stage('prod', promoting)], [])).toBe(true);
  });

  test('a Stage promoting through a PromotionRequest is no reason to read rollouts', () => {
    expect(
      shouldPollRollouts(
        [
          stage('prod', {
            ...promoting,
            currentPromotionRequest: { name: 'prod.01abc.deadbeef' }
          })
        ],
        []
      )
    ).toBe(false);
  });
});

describe('recordedRollouts()', () => {
  test('changes when a Stage records another rollout', () => {
    const before = recordedRollouts([
      stage('prod', { lastPromotionRollout: { name: 'prod.rollout-1' } }),
      stage('staging')
    ]);
    const after = recordedRollouts([
      stage('prod', { lastPromotionRollout: { name: 'prod.rollout-2' } }),
      stage('staging')
    ]);
    expect(before).not.toBe(after);
  });

  test('does not depend on the order the Stages come in', () => {
    const prod = stage('prod', { lastPromotionRollout: { name: 'prod.rollout-1' } });
    const staging = stage('staging', { lastPromotionRollout: { name: 'staging.rollout-4' } });
    expect(recordedRollouts([prod, staging])).toBe(recordedRollouts([staging, prod]));
  });

  test('is unmoved by anything else a Stage reports', () => {
    const recorded = { lastPromotionRollout: { name: 'prod.rollout-1' } };
    expect(recordedRollouts([stage('prod', recorded)])).toBe(
      recordedRollouts([stage('prod', { ...recorded, ...promoting })])
    );
  });
});
