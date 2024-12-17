// the UI space for plugins but with limit in terms of layout

import { DeepLinkPromotion } from './deep-link-promotion';
import { DeepLinkPromotionStep } from './deep-link-promotion-step';

export const UiPluginHoles = {
  DeepLinks: {
    PromotionStep: DeepLinkPromotionStep,
    Promotion: DeepLinkPromotion
  }
};
