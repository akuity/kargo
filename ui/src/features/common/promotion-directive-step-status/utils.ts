import { PromotionStatusPhase } from '@ui/features/common/promotion-status/utils';
import { PromotionStatus } from '@ui/gen/v1alpha1/generated_pb';

// UI concludes from Promotion's status data
export enum PromotionDirectiveStepStatus {
  RUNNING,
  FAILED,
  SUCCESS,
  WONT_RUN // because previous step failed
}

export const getPromotionDirectiveStepStatus = (
  stepNumber: number,
  promotionStatus?: PromotionStatus
) => {
  let currentStep = 0;

  // there will always be some value of currentStep
  // this is just for type saftey
  if (promotionStatus?.currentStep) {
    currentStep = Number(promotionStatus.currentStep);
  }

  if (stepNumber < currentStep) {
    // assumes that controller successfully ran this step
    return PromotionDirectiveStepStatus.SUCCESS;
  }

  if (stepNumber === currentStep) {
    switch (promotionStatus?.phase) {
      case PromotionStatusPhase.RUNNING:
        return PromotionDirectiveStepStatus.RUNNING;
      case PromotionStatusPhase.ERRORED:
      case PromotionStatusPhase.FAILED:
        return PromotionDirectiveStepStatus.FAILED;
      case PromotionStatusPhase.SUCCEEDED:
        return PromotionDirectiveStepStatus.SUCCESS;
    }
  }

  // if step number is > current step, no matter which state promotion is in, the given step number "has not" run yet or "will not" depends on the promotion phase but that doesn't matter....YET
  return PromotionDirectiveStepStatus.WONT_RUN;
};
