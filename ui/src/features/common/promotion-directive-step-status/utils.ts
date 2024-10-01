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
  if (
    promotionStatus?.phase === PromotionStatusPhase.RUNNING &&
    stepNumber === Number(promotionStatus?.currentStep)
  ) {
    return PromotionDirectiveStepStatus.RUNNING;
  }

  if (
    promotionStatus?.phase === PromotionStatusPhase.ERRORED &&
    stepNumber === Number(promotionStatus?.currentStep)
  ) {
    return PromotionDirectiveStepStatus.FAILED;
  }

  if (
    promotionStatus?.phase === PromotionStatusPhase.ERRORED &&
    stepNumber > Number(promotionStatus?.currentStep)
  ) {
    return PromotionDirectiveStepStatus.WONT_RUN;
  }

  return PromotionDirectiveStepStatus.SUCCESS;
};
