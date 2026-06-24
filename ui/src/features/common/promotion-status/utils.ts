import { Promotion } from '@ui/gen/api/v2/models';

// TODO: can we map this to promotion_types.go?
export enum PromotionStatusPhase {
  PENDING = 'Pending',
  RUNNING = 'Running',
  SUCCEEDED = 'Succeeded',
  FAILED = 'Failed',
  ERRORED = 'Errored',
  ABORTED = 'Aborted'
}

export const getPromotionStatusPhase = (promotion: Promotion) =>
  promotion?.status?.phase as PromotionStatusPhase;

// backend equivalent logic - read in promotion_types.go
export const isPromotionPhaseTerminal = (promotionPhase: PromotionStatusPhase) => {
  switch (promotionPhase) {
    case PromotionStatusPhase.SUCCEEDED:
    case PromotionStatusPhase.FAILED:
    case PromotionStatusPhase.ERRORED:
    case PromotionStatusPhase.ABORTED:
      return true;
  }

  return false;
};

export const isPromotionRetryable = (phase: PromotionStatusPhase) => {
  switch (phase) {
    case PromotionStatusPhase.FAILED:
    case PromotionStatusPhase.ERRORED:
    case PromotionStatusPhase.ABORTED:
      return true;
  }

  return false;
};
