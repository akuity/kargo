import { HealthCheckStep, Promotion, PromotionStep } from '@ui/gen/api/v2/models';

export const getPromotionStepConfig = (step: PromotionStep): Record<string, unknown> =>
  (step?.config as Record<string, unknown>) || {};

export const getPromotionState = (promotion: Promotion): Record<string, unknown> =>
  (promotion?.status?.state as Record<string, unknown>) || {};

export const getPromotionHealthCheckConfig = (hc: HealthCheckStep): Record<string, unknown> =>
  (hc?.config as Record<string, unknown>) || {};

export const getPromotionStepAlias = (promotionStep: PromotionStep, stepIndex: string | number) =>
  promotionStep?.as || `step-${stepIndex}`;
