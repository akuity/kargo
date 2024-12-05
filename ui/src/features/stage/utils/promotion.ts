// action, reaction and behaviour of everything related to promotion

import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal
} from '@ui/features/common/promotion-status/utils';
import { Promotion } from '@ui/gen/v1alpha1/generated_pb';
import { k8sApiMachineryTimestampDate } from '@ui/utils/connectrpc-extension';

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

export const promotionCompareFn = (
  promotion1: Partial<Promotion>,
  promotion2: Partial<Promotion>
) => {
  const promo1Date = k8sApiMachineryTimestampDate(promotion1.metadata?.creationTimestamp);
  const promo2Date = k8sApiMachineryTimestampDate(promotion2.metadata?.creationTimestamp);

  if (promo1Date && promo2Date) {
    // latest promotion should have lower index in array

    if (promo2Date < promo1Date) {
      return -1;
    }

    if (promo1Date < promo2Date) {
      return 1;
    }
  }

  // doesn't matter that much... this is to keep UI in the same state on refresh because dates are in second precision and the promotion that happened in same seconds needs ordering
  return (promotion1?.metadata?.name || 0) < (promotion2?.metadata?.name || 0) ? -1 : 1;
};
