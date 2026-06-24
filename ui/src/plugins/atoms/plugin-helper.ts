import { HealthCheckStep, Promotion, PromotionStep } from '@ui/gen/api/v2/models';

export const getPromotionStepConfig = (step: PromotionStep): Record<string, unknown> =>
  step?.config || {};

export const getPromotionState = (promotion: Promotion): Record<string, unknown> =>
  promotion?.status?.state || {};

export const getPromotionHealthCheckConfig = (hc: HealthCheckStep): Record<string, unknown> =>
  hc?.config || {};

export const getPromotionStepAlias = (promotionStep: PromotionStep, stepIndex: string | number) =>
  promotionStep?.as || `step-${stepIndex}`;
