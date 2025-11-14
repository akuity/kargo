import { PromotionStatus } from '@ui/gen/api/v1alpha1/generated_pb';

import { PromotionStepStatus } from './promotion-step-status';

// UI concludes from Promotion's status data
export enum PromotionDirectiveStepStatus {
  RUNNING,
  FAILED,
  SUCCESS,
  SKIPPED,
  WONT_RUN // because previous step failed
}

export const getPromotionDirectiveStepStatus = (
  stepNumber: number,
  promotionStatus?: PromotionStatus
) => {
  const promotionStepStatus = promotionStatus?.stepExecutionMetadata?.[stepNumber]
    ?.status as PromotionStepStatus;

  switch (promotionStepStatus) {
    case PromotionStepStatus.RUNNING:
      return PromotionDirectiveStepStatus.RUNNING;
    case PromotionStepStatus.SKIPPED:
      return PromotionDirectiveStepStatus.SKIPPED;
    case PromotionStepStatus.SUCCEEDED:
      return PromotionDirectiveStepStatus.SUCCESS;
    case PromotionStepStatus.ABORTED:
    case PromotionStepStatus.ERRORED:
    case PromotionStepStatus.FAILED:
      return PromotionDirectiveStepStatus.FAILED;
  }

  return PromotionDirectiveStepStatus.WONT_RUN;
};
