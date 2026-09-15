import { useSyncExternalStore } from 'react';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import { isPromotionWindowOpen } from './is-promotion-window-open';
import { toViewerClockMs } from './viewer-clock';

const MINUTE_MS = 60_000;

const subscribeToMinute = (onMinuteChange: () => void) => {
  const interval = setInterval(onMinuteChange, MINUTE_MS);
  return () => clearInterval(interval);
};

const currentMinute = () => Math.floor(Date.now() / MINUTE_MS);

export const useIsPromotionWindowOpen = (promotionWindows: PromotionWindow[]) => {
  const minute = useSyncExternalStore(subscribeToMinute, currentMinute);

  return isPromotionWindowOpen(promotionWindows, toViewerClockMs(minute * MINUTE_MS));
};
