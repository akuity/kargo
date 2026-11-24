import { describe, expect, test } from 'vitest';

import { Stage, V1Condition } from '@ui/gen/api/v2/models';

import { getStagePhase, getStageStatus, phaseTone, StagePhase } from './utils';

const stageWith = (...conditions: V1Condition[]): Stage => ({ status: { conditions } }) as Stage;

const condition = (type: string, status: string, extra: Partial<V1Condition> = {}) =>
  ({ type, status, ...extra }) as V1Condition;

const promoting = condition('Promoting', 'True');
const verifying = condition('Verified', 'Unknown');
const reconciling = condition('Reconciling', 'True');
const ready = condition('Ready', 'True');
const failed = condition('Ready', 'False');

describe('getStageStatus()', () => {
  test('reports Unknown for a Stage carrying no conditions', () => {
    expect(getStageStatus(stageWith())).toEqual({ phase: 'Unknown', message: '' });
  });

  test('reports the message the deciding condition carried', () => {
    const stage = stageWith(condition('Ready', 'False', { message: 'health check failed' }));
    expect(getStageStatus(stage)).toEqual({ phase: 'Failed', message: 'health check failed' });
  });

  test.each([
    ['Promoting', promoting],
    ['Verifying', verifying],
    ['Reconciling', reconciling],
    ['Ready', ready],
    ['Failed', failed]
  ])('reports %s', (phase, only) => {
    expect(getStageStatus(stageWith(only)).phase).toBe(phase);
  });

  // Whichever conditions are set at once, the first in this order wins.
  test('holds the priority order Promoting > Verifying > Reconciling > Failed > Ready', () => {
    const inOrder = [promoting, verifying, reconciling, failed];
    const expected: StagePhase[] = ['Promoting', 'Verifying', 'Reconciling', 'Failed'];

    inOrder.forEach((_, i) => {
      // every condition from this one down, so the one under test is the first
      expect(getStageStatus(stageWith(...inOrder.slice(i))).phase).toBe(expected[i]);
    });

    expect(getStageStatus(stageWith(ready)).phase).toBe('Ready');
  });

  // NoFreight stands a condition down rather than asserting anything, so a
  // Stage with nothing to promote reads as Unknown however its conditions look.
  test.each([
    ['Promoting', 'True'],
    ['Verified', 'Unknown'],
    ['Ready', 'False']
  ])('passes over a %s stood down by NoFreight', (type, status) => {
    const stage = stageWith(condition(type, status, { reason: 'NoFreight', message: 'ignored' }));
    expect(getStageStatus(stage)).toEqual({ phase: 'Unknown', message: '' });
  });

  // Reconciling carries no such guard, and did not before.
  test('honours Reconciling whatever its reason', () => {
    const stage = stageWith(condition('Reconciling', 'True', { reason: 'NoFreight' }));
    expect(getStageStatus(stage).phase).toBe('Reconciling');
  });

  // Those conditions were written by the controller that has since gone quiet,
  // so none of them can be trusted, however healthy they read.
  test('reports Failed for a dead controller, whatever the conditions say', () => {
    const stage = stageWith(condition('Ready', 'True', { message: 'all good' }));
    expect(getStageStatus(stage, true)).toEqual({ phase: 'Failed', message: '' });
  });
});

describe('getStagePhase()', () => {
  test('is the phase getStageStatus reports', () => {
    const stage = stageWith(verifying);
    expect(getStagePhase(stage)).toBe(getStageStatus(stage).phase);
    expect(getStagePhase(stage, true)).toBe('Failed');
  });
});

describe('phaseTone', () => {
  // A phase with no tone would be stated in whatever color it inherited.
  test('gives every phase a tone', () => {
    const phases: StagePhase[] = [
      'Promoting',
      'Verifying',
      'Reconciling',
      'Failed',
      'Ready',
      'Unknown'
    ];
    phases.forEach((phase) => expect(phaseTone[phase]).toBeTruthy());
  });
});
