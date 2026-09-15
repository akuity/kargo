import { PromotionWindow, PromotionWindowKind } from '@ui/gen/api/v2/models';

import { parsePromotionWindows } from './parse-promotion-windows';

export const isPromotionWindowOpen = (promotionWindows: PromotionWindow[], at: number): boolean => {
  const enabledWindows = promotionWindows.filter((w) => !w.disabled);

  const occurrences = parsePromotionWindows(enabledWindows, { before: new Date(at) }).sort(
    (a, b) => a.start.getTime() - b.start.getTime()
  );

  const hasExplicitAllow = enabledWindows.some(
    (w) => w.kind === PromotionWindowKind.PromotionWindowKindAllow
  );

  let allow = false;
  let deny = false;

  for (const occurrence of occurrences) {
    if (at >= occurrence.start.getTime() && at < occurrence.end.getTime()) {
      if (occurrence.kind === PromotionWindowKind.PromotionWindowKindAllow) {
        allow = true;
      }
      if (occurrence.kind === PromotionWindowKind.PromotionWindowKindDeny) {
        deny = true;
      }
    }
  }

  if (deny) {
    return false;
  }

  if (allow) {
    return true;
  }

  if (hasExplicitAllow) {
    return false;
  }

  return true;
};
