import { describe, expect, test } from 'vitest';

import { InflatedPromotionTask, PromotionStep } from '@ui/gen/api/v2/models';

import { PromotionDirectiveStepStatus } from '../../common/promotion-directive-step-status/utils';

import {
  countStepStatuses,
  didPromotionEndUnsuccessfullyInTaskGroup,
  getEffectiveVars,
  getTaskGroupExecutionMetadata,
  getTaskGroupStatus,
  getTaskGroupStepStatuses,
  getUserStepAlias,
  groupPromotionSteps,
  PromotionTaskGroup
} from './promotion-step-groups';

const inflatedTask = (alias: string, steps: string[]): InflatedPromotionTask => ({
  alias,
  task: { kind: 'PromotionTask', name: 'render' },
  steps
});

const taskStep = (as: string): PromotionStep => ({ as, uses: 'git-clone' });

const taskGroup = (steps: PromotionStep[]): PromotionTaskGroup => ({
  type: 'task',
  key: 'task-1',
  inflatedTask: inflatedTask(
    'task-1',
    steps.map((step) => step.as ?? '')
  ),
  steps: steps.map((step, index) => ({ type: 'step', step, index }))
});

const { FAILED, RETRYING, RUNNING, SKIPPED, SUCCESS, WONT_RUN } = PromotionDirectiveStepStatus;

describe('getUserStepAlias()', () => {
  test.each([
    { name: 'no alias', alias: undefined, expected: undefined },
    { name: 'generated step alias', alias: 'step-3', expected: undefined },
    { name: 'generated task step alias', alias: 'task-7::step-2', expected: undefined },
    { name: 'user alias', alias: 'push', expected: 'push' },
    { name: 'user alias inside a task', alias: 'task-7::commit', expected: 'commit' },
    {
      name: 'generated step in a user-named task',
      alias: 'render-prod::step-1',
      expected: undefined
    }
  ])('$name', ({ alias, expected }) => {
    expect(getUserStepAlias(alias)).toBe(expected);
  });
});

describe('groupPromotionSteps()', () => {
  test('keeps steps as plain steps when no task was inflated', () => {
    const steps = [{ as: 'step-1' }, { as: 'step-2' }];
    expect(groupPromotionSteps(steps)).toEqual([
      { type: 'step', step: steps[0], index: 0 },
      { type: 'step', step: steps[1], index: 1 }
    ]);
  });

  test('groups the steps inflated from a task', () => {
    const steps = [taskStep('task-1::step-1'), taskStep('task-1::commit')];
    const task = inflatedTask('task-1', ['task-1::step-1', 'task-1::commit']);
    expect(groupPromotionSteps(steps, [task])).toEqual([
      {
        type: 'task',
        key: 'task-1',
        inflatedTask: task,
        steps: [
          { type: 'step', step: steps[0], index: 0 },
          { type: 'step', step: steps[1], index: 1 }
        ]
      }
    ]);
  });

  test('splits back-to-back instances of the same task', () => {
    const steps = [taskStep('task-1::step-1'), taskStep('task-2::step-1')];
    const tasks = [
      inflatedTask('task-1', ['task-1::step-1']),
      inflatedTask('task-2', ['task-2::step-1'])
    ];
    expect(
      groupPromotionSteps(steps, tasks).map((entry) => entry.type === 'task' && entry.key)
    ).toEqual(['task-1', 'task-2']);
  });

  test('keeps plain steps between task groups in spec order', () => {
    const steps = [taskStep('task-1::step-1'), { as: 'step-2' }, taskStep('task-3::step-1')];
    const tasks = [
      inflatedTask('task-1', ['task-1::step-1']),
      inflatedTask('task-3', ['task-3::step-1'])
    ];
    expect(groupPromotionSteps(steps, tasks).map((entry) => entry.type)).toEqual([
      'task',
      'step',
      'task'
    ]);
  });

  test('treats a user alias that looks like a task step as a plain step', () => {
    const steps = [taskStep('task-1::step-1'), { as: 'task-1::push' }];
    const tasks = [inflatedTask('task-1', ['task-1::step-1'])];
    expect(groupPromotionSteps(steps, tasks).map((entry) => entry.type)).toEqual(['task', 'step']);
  });
});

describe('getTaskGroupStepStatuses()', () => {
  test('reads each step status by its spec index', () => {
    const group = taskGroup([taskStep('task-1::step-1'), taskStep('task-1::step-2')]);
    expect(
      getTaskGroupStepStatuses(group, {
        phase: 'Running',
        currentStep: 1,
        stepExecutionMetadata: [{ status: 'Succeeded' }, { status: 'Running' }]
      })
    ).toEqual([SUCCESS, RUNNING]);
  });
});

describe('getTaskGroupStatus()', () => {
  test.each([
    { name: 'failed wins', statuses: [SUCCESS, RETRYING, FAILED], expected: FAILED },
    { name: 'retrying over running', statuses: [SUCCESS, RETRYING, RUNNING], expected: RETRYING },
    { name: 'running', statuses: [SUCCESS, RUNNING, WONT_RUN], expected: RUNNING },
    { name: 'all skipped', statuses: [SKIPPED, SKIPPED], expected: SKIPPED },
    { name: 'not started', statuses: [WONT_RUN, WONT_RUN], expected: WONT_RUN },
    { name: 'partially done', statuses: [SUCCESS, WONT_RUN], expected: RUNNING },
    { name: 'succeeded with skips', statuses: [SUCCESS, SKIPPED], expected: SUCCESS }
  ])('$name', ({ statuses, expected }) => {
    expect(getTaskGroupStatus(statuses)).toBe(expected);
  });

  test('promotion ended unsuccessfully in the group', () => {
    expect(getTaskGroupStatus([SUCCESS, WONT_RUN], true)).toBe(FAILED);
  });
});

describe('didPromotionEndUnsuccessfullyInTaskGroup()', () => {
  const group: PromotionTaskGroup = {
    ...taskGroup([taskStep('task-1::step-1'), taskStep('task-1::step-2')]),
    steps: [
      { type: 'step', step: taskStep('task-1::step-1'), index: 1 },
      { type: 'step', step: taskStep('task-1::step-2'), index: 2 }
    ]
  };

  test.each([
    { name: 'no status', promotionStatus: undefined, expected: false },
    {
      name: 'still running',
      promotionStatus: { phase: 'Running', currentStep: 2 },
      expected: false
    },
    { name: 'succeeded', promotionStatus: { phase: 'Succeeded', currentStep: 2 }, expected: false },
    {
      name: 'ended before the group',
      promotionStatus: { phase: 'Errored', currentStep: 0 },
      expected: false
    },
    {
      name: 'ended after the group',
      promotionStatus: { phase: 'Failed', currentStep: 3 },
      expected: false
    },
    {
      name: 'errored in the group',
      promotionStatus: { phase: 'Errored', currentStep: 2 },
      expected: true
    },
    {
      name: 'aborted in the group',
      promotionStatus: { phase: 'Aborted', currentStep: 1 },
      expected: true
    }
  ])('$name', ({ promotionStatus, expected }) => {
    expect(didPromotionEndUnsuccessfullyInTaskGroup(group, promotionStatus)).toBe(expected);
  });
});

describe('countStepStatuses()', () => {
  test('counts each status', () => {
    expect(countStepStatuses([SUCCESS, SUCCESS, SKIPPED])).toEqual({
      [SUCCESS]: 2,
      [SKIPPED]: 1
    });
  });
});

describe('getTaskGroupExecutionMetadata()', () => {
  const group = taskGroup([
    taskStep('task-1::step-1'),
    taskStep('task-1::step-2'),
    taskStep('task-1::step-3')
  ]);

  test('spans from the first start to the last finish', () => {
    expect(
      getTaskGroupExecutionMetadata(group, {
        stepExecutionMetadata: [
          { startedAt: '2026-10-22T00:00:00Z', finishedAt: '2026-10-22T00:00:05Z' },
          { startedAt: '2026-10-22T00:00:05Z', finishedAt: '2026-10-22T00:00:09Z' },
          { status: 'Skipped' }
        ]
      })
    ).toEqual({ startedAt: '2026-10-22T00:00:00Z', finishedAt: '2026-10-22T00:00:09Z' });
  });

  test('is empty when no step has run', () => {
    expect(getTaskGroupExecutionMetadata(group, {})).toEqual({
      startedAt: undefined,
      finishedAt: undefined
    });
  });
});

describe('getEffectiveVars()', () => {
  test.each([
    { name: 'no vars', vars: undefined, expected: [] },
    {
      name: 'unique names',
      vars: [
        { name: 'branch', value: 'main' },
        { name: 'path', value: 'env/prod' }
      ],
      expected: [
        { name: 'branch', value: 'main' },
        { name: 'path', value: 'env/prod' }
      ]
    },
    {
      name: 'step var overrides task default in place',
      vars: [
        { name: 'branch', value: 'main' },
        { name: 'path', value: 'env/prod' },
        { name: 'branch', value: 'stage/prod' }
      ],
      expected: [
        { name: 'branch', value: 'stage/prod' },
        { name: 'path', value: 'env/prod' }
      ]
    }
  ])('$name', ({ vars, expected }) => {
    expect(getEffectiveVars(vars)).toEqual(expected);
  });
});
