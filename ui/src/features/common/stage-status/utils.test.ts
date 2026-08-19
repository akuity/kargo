import { describe, expect, test } from 'vitest';

import type { Stage } from '@ui/gen/api/v2/models';

import { getStagePhase, isStageProgressing } from './utils';

const stageWithConditions = (conditions: NonNullable<Stage['status']>['conditions']): Stage =>
  ({ status: { conditions } }) as Stage;

describe('stage status', () => {
  test('reports a progressing Stage instead of failed', () => {
    const stage = stageWithConditions([{ type: 'Ready', status: 'False', reason: 'Progressing' }]);

    expect(isStageProgressing(stage)).toBe(true);
    expect(getStagePhase(stage)).toBe('Progressing');
  });

  test('keeps terminal Stage failures failed', () => {
    const stage = stageWithConditions([
      { type: 'Ready', status: 'False', reason: 'VerificationFailed' }
    ]);

    expect(isStageProgressing(stage)).toBe(false);
    expect(getStagePhase(stage)).toBe('Failed');
  });
});
