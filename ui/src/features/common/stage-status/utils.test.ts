import { describe, expect, test } from 'vitest';

import type { Stage, V1Condition } from '@ui/gen/api/v2/models';

import { getStagePhase } from './utils';

const condition = (type: string, status: string, reason: string): V1Condition => ({
  type,
  status,
  reason,
  message: '',
  lastTransitionTime: ''
});

const stage = (conditions: V1Condition[]): Stage => ({ status: { conditions } }) as Stage;

describe('getStagePhase', () => {
  test('Failed when the controller is dead', () => {
    expect(getStagePhase(stage([]), true)).toEqual('Failed');
  });

  test('Promoting when a promotion is in progress', () => {
    expect(getStagePhase(stage([condition('Promoting', 'True', 'ActivePromotion')]))).toEqual(
      'Promoting'
    );
  });

  test('Verifying when verification is in progress', () => {
    expect(getStagePhase(stage([condition('Verified', 'Unknown', 'VerificationRunning')]))).toEqual(
      'Verifying'
    );
  });

  test('Reconciling when the controller is retrying after an error', () => {
    expect(getStagePhase(stage([condition('Reconciling', 'True', 'RetryAfterError')]))).toEqual(
      'Reconciling'
    );
  });

  test('Progressing, not Failed, when not Ready because health is progressing', () => {
    expect(
      getStagePhase(
        stage([
          condition('Ready', 'False', 'Progressing'),
          condition('Healthy', 'Unknown', 'Progressing'),
          condition('Verified', 'True', 'Verified')
        ])
      )
    ).toEqual('Progressing');
  });

  test('Progressing when waiting for a post-promotion health check', () => {
    expect(getStagePhase(stage([condition('Ready', 'False', 'WaitingForHealthCheck')]))).toEqual(
      'Progressing'
    );
  });

  test('Failed when not Ready because the Stage is unhealthy', () => {
    expect(getStagePhase(stage([condition('Ready', 'False', 'Unhealthy')]))).toEqual('Failed');
  });

  test('Failed when not Ready because the last promotion failed', () => {
    expect(getStagePhase(stage([condition('Ready', 'False', 'LastPromotionFailed')]))).toEqual(
      'Failed'
    );
  });

  test('Unknown when not Ready due to having no Freight', () => {
    expect(getStagePhase(stage([condition('Ready', 'False', 'NoFreight')]))).toEqual('Unknown');
  });

  test('Ready when the Ready condition is True', () => {
    expect(getStagePhase(stage([condition('Ready', 'True', 'Verified')]))).toEqual('Ready');
  });

  test('Unknown when there are no conditions', () => {
    expect(getStagePhase(stage([]))).toEqual('Unknown');
  });
});
