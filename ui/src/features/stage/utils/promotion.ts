// action, reaction and behaviour of everything related to promotion

import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal
} from '@ui/features/common/promotion-status/utils';
import { Promotion } from '@ui/gen/v1alpha1/generated_pb';

export const canAbortPromotion = (promotion: Promotion) =>
  !isPromotionPhaseTerminal(getPromotionStatusPhase(promotion));

// API annotates promotion metadata to let controller abort promotion
export const hasAbortRequest = (promotion: Promotion) => {
  const abortAnnotation =
    promotion.metadata?.annotations[
      // as this hard-coded annotation/labels increase, put it all at one place
      'kargo.akuity.io/abort'
    ];

  return !!abortAnnotation;
};
