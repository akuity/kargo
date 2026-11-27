import { ApiError } from '@ui/lib/api/custom-fetch';

export const createdPromotionMaxRetries = 5;

const createdPromotionBaseRetryDelayMs = 250;
const createdPromotionMaxRetryDelayMs = 2000;

type CreatedPromotionState = { createdPromotion?: string };

export const createdPromotionState = (promotion: string | undefined): CreatedPromotionState => ({
  createdPromotion: promotion
});

export const isCreatedPromotion = (locationState: unknown, promotion: string) =>
  !!promotion && (locationState as CreatedPromotionState | null)?.createdPromotion === promotion;

export const shouldRetryCreatedPromotion = (failureCount: number, error: unknown) =>
  failureCount < createdPromotionMaxRetries && error instanceof ApiError && error.isNotFound();

export const createdPromotionRetryDelay = (failureCount: number) =>
  Math.min(createdPromotionBaseRetryDelayMs * 2 ** failureCount, createdPromotionMaxRetryDelayMs);
