import { formatDistance } from 'date-fns';

import type { Stage } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

const defaultClosedReason = 'A promotion window currently forbids promotion of this Stage.';

export const isPromotionWindowClosed = (stage?: Stage): boolean =>
  Boolean(stage?.status?.promotionWindowStatus?.closed);

export const promotionWindowClosedReason = (stage?: Stage): string => {
  const reason = (stage?.status?.promotionWindowStatus?.reason || defaultClosedReason).trim();
  return /[.!?]$/.test(reason) ? reason : `${reason}.`;
};

export const promotionWindowNextOpen = (stage?: Stage): Date | undefined =>
  parseDate(stage?.status?.promotionWindowStatus?.nextOpen);

export const promotionWindowClosedMessage = (stage?: Stage, now: Date = new Date()): string => {
  if (!isPromotionWindowClosed(stage)) {
    return '';
  }

  const reason = promotionWindowClosedReason(stage);
  const nextOpen = promotionWindowNextOpen(stage);

  if (!nextOpen) {
    return `${reason} No reopening time is known.`;
  }

  return `${reason} Promotion reopens ${formatDistance(nextOpen, now, { addSuffix: true })}.`;
};

export const promotionWindowReopensLabel = (stage?: Stage, now: Date = new Date()): string => {
  if (!isPromotionWindowClosed(stage)) {
    return '';
  }

  const nextOpen = promotionWindowNextOpen(stage);

  return nextOpen
    ? `Reopens ${formatDistance(nextOpen, now, { addSuffix: true })}`
    : 'No known reopening time';
};
