import { expect, test } from 'vitest';

import { PromotionStatus } from '@ui/gen/api/v1alpha1/generated_pb';
import { PlainMessage } from '@ui/utils/connectrpc-utils';

import { PromotionStatusPhase } from '../promotion-status/utils';

import { getPromotionDirectiveStepStatus, PromotionDirectiveStepStatus } from './utils';

test('getPromotionDirectiveStepStatus', () => {
  const promotionStatus = (phase: PromotionStatusPhase, currentStep: number) =>
    ({
      phase,
      currentStep: BigInt(currentStep)
    }) satisfies Partial<PlainMessage<PromotionStatus>> as PromotionStatus;

  const suite: Record<
    PromotionDirectiveStepStatus,
    Array<{
      step: number;
      promotionStatus: {
        currentStep: number;
        phase: PromotionStatusPhase;
      };
    }>
  > = {
    [PromotionDirectiveStepStatus.RUNNING]: [
      {
        step: 0,
        promotionStatus: {
          phase: PromotionStatusPhase.RUNNING,
          currentStep: 0
        }
      },
      {
        step: 100000,
        promotionStatus: {
          phase: PromotionStatusPhase.RUNNING,
          currentStep: 100000
        }
      }
    ],
    [PromotionDirectiveStepStatus.FAILED]: [
      {
        step: 5,
        promotionStatus: {
          phase: PromotionStatusPhase.FAILED,
          currentStep: 5
        }
      },
      {
        step: 111,
        promotionStatus: {
          phase: PromotionStatusPhase.FAILED,
          currentStep: 111
        }
      },
      {
        step: 1353,
        promotionStatus: {
          phase: PromotionStatusPhase.ERRORED,
          currentStep: 1353
        }
      },
      {
        step: 1241243,
        promotionStatus: {
          phase: PromotionStatusPhase.FAILED,
          currentStep: 1241243
        }
      }
    ],
    [PromotionDirectiveStepStatus.SUCCESS]: [
      {
        step: 1,
        promotionStatus: {
          phase: PromotionStatusPhase.SUCCEEDED,
          currentStep: 100
        }
      },
      {
        step: 1,
        promotionStatus: {
          phase: PromotionStatusPhase.SUCCEEDED,
          currentStep: 1
        }
      },
      {
        step: 1,
        promotionStatus: {
          phase: PromotionStatusPhase.RUNNING,
          currentStep: 100
        }
      },
      {
        step: 123123,
        promotionStatus: {
          phase: PromotionStatusPhase.RUNNING,
          currentStep: 123124
        }
      },
      {
        step: 12,
        promotionStatus: {
          phase: PromotionStatusPhase.FAILED,
          currentStep: 1231
        }
      },
      {
        step: 12,
        promotionStatus: {
          phase: PromotionStatusPhase.ERRORED,
          currentStep: 15
        }
      },
      {
        step: 11,
        promotionStatus: {
          phase: PromotionStatusPhase.ABORTED,
          currentStep: 15
        }
      }
    ],
    [PromotionDirectiveStepStatus.WONT_RUN]: [
      {
        step: 15,
        promotionStatus: {
          currentStep: 10,
          phase: PromotionStatusPhase.ERRORED
        }
      },
      {
        step: 15,
        promotionStatus: {
          currentStep: 10,
          phase: PromotionStatusPhase.RUNNING
        }
      },
      {
        step: 1111,
        promotionStatus: {
          currentStep: 111,
          phase: PromotionStatusPhase.FAILED
        }
      },
      {
        step: 20,
        promotionStatus: {
          currentStep: 2,
          phase: PromotionStatusPhase.PENDING
        }
      },
      {
        step: 0,
        promotionStatus: {
          currentStep: 0,
          phase: PromotionStatusPhase.PENDING
        }
      },
      {
        step: 0,
        promotionStatus: {
          currentStep: 0,
          phase: PromotionStatusPhase.ABORTED
        }
      },
      {
        step: 100,
        promotionStatus: {
          currentStep: 90,
          phase: PromotionStatusPhase.ABORTED
        }
      }
    ]
  };

  for (const [expectedStatus, testCases] of Object.entries(suite)) {
    for (const testCase of testCases) {
      expect(
        PromotionDirectiveStepStatus[
          getPromotionDirectiveStepStatus(
            testCase.step,
            promotionStatus(testCase.promotionStatus.phase, testCase.promotionStatus.currentStep)
          )
        ],
        // @ts-expect-error really not sure why but its working...
        `when promotion phase is in ${testCase.promotionStatus.phase} state and promotion current step is ${testCase.promotionStatus.currentStep}, step ${testCase.step} must be in ${PromotionDirectiveStepStatus[expectedStatus]}`
      ).toBe(PromotionDirectiveStepStatus[+expectedStatus]);
    }
  }
});
