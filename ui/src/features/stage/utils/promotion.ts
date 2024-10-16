// action, reaction and behaviour of everything related to promotion

import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal
} from '@ui/features/common/promotion-status/utils';
import { Promotion } from '@ui/gen/v1alpha1/generated_pb';

export const canAbortPromotion = (promotion: Promotion) =>
  !isPromotionPhaseTerminal(getPromotionStatusPhase(promotion));
