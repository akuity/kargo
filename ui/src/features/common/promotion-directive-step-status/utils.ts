import { PromotionStatus } from '@ui/gen/api/v2/models';

import { isPromotionPhaseTerminal, PromotionStatusPhase } from '../promotion-status/utils';

import { PromotionStepStatus } from './promotion-step-status';

// UI concludes from Promotion's status data
export enum PromotionDirectiveStepStatus {
  RUNNING,
  FAILED,
  SUCCESS,
  SKIPPED,
  WONT_RUN, // because previous step failed
  RETRYING
}

const retryableStepStatuses: string[] = [PromotionStepStatus.ERRORED, PromotionStepStatus.FAILED];

const isRetryPending = (stepNumber: number, promotionStatus?: PromotionStatus) => {
  if (!promotionStatus) {
    return false;
  }

  if (isPromotionPhaseTerminal(promotionStatus.phase as PromotionStatusPhase)) {
    return false;
  }

  if ((promotionStatus.currentStep ?? 0) !== stepNumber) {
    return false;
  }

  const stepExecutionMetadata = promotionStatus.stepExecutionMetadata?.[stepNumber];

  if (!stepExecutionMetadata) {
    return false;
  }

  return (
    retryableStepStatuses.includes(stepExecutionMetadata.status ?? '') &&
    !!stepExecutionMetadata.startedAt &&
    !stepExecutionMetadata.finishedAt &&
    (stepExecutionMetadata.errorCount ?? 0) > 0
  );
};

export const getStepErrorCount = (stepNumber: number, promotionStatus?: PromotionStatus) =>
  promotionStatus?.stepExecutionMetadata?.[stepNumber]?.errorCount ?? 0;

export const getPromotionDirectiveStepStatus = (
  stepNumber: number,
  promotionStatus?: PromotionStatus
) => {
  if (isRetryPending(stepNumber, promotionStatus)) {
    return PromotionDirectiveStepStatus.RETRYING;
  }

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

export const isFailedStep = (stepIndex: number, promotionStatus?: PromotionStatus) =>
  getPromotionDirectiveStepStatus(stepIndex, promotionStatus) ===
  PromotionDirectiveStepStatus.FAILED;

export const isRetryingStep = (stepIndex: number, promotionStatus?: PromotionStatus) =>
  getPromotionDirectiveStepStatus(stepIndex, promotionStatus) ===
  PromotionDirectiveStepStatus.RETRYING;

export const isProgressingStep = (stepIndex: number, promotionStatus?: PromotionStatus) => {
  const status = getPromotionDirectiveStepStatus(stepIndex, promotionStatus);
  return (
    status === PromotionDirectiveStepStatus.RUNNING ||
    status === PromotionDirectiveStepStatus.RETRYING
  );
};
