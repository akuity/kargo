import { PromotionStatusPhase } from '@ui/features/common/promotion-status/utils';
import { PromotionStatus } from '@ui/gen/v1alpha1/generated_pb';

// UI concludes from Promotion's status data
export enum PromotionDirectiveStepStatus {
  RUNNING,
  FAILED,
  SUCCESS
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

  // TODO: controller should point 'currentStep' to exact failed step
  // at the moment, it is defaulted to 0
  if (promotionStatus?.phase !== PromotionStatusPhase.SUCCEEDED) {
    return PromotionDirectiveStepStatus.FAILED;
  }

  return PromotionDirectiveStepStatus.SUCCESS;
};
