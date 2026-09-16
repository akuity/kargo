import { addDays, endOfDay, startOfMonth, startOfWeek } from 'date-fns';
import { useMemo } from 'react';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import { parsePromotionWindows } from './parse-promotion-windows';
import { toViewerClockDate } from './viewer-clock';

/**
 * The calendar always lays its month out as a fixed six week grid, so it can
 * render well past the end of the month -- up to two extra weeks when the
 * month both begins on the first day of a week and is short.
 */
const CALENDAR_GRID_DAYS = 42;

/** Every day the calendar renders for the month containing `date`. */
export const visibleRange = (date: Date) => {
  const from = startOfWeek(startOfMonth(date));
  const to = endOfDay(addDays(from, CALENDAR_GRID_DAYS - 1));

  // IMPORTANT - BEFORE YOU GIVE RRULE TO QUERY, YOU NEED TO STAMP THE TIMEZONE OFFSET IT IS IN
  return { from: toViewerClockDate(from), to: toViewerClockDate(to) };
};

export const useGetPromotionWindowOccurrences = (promotionWindows: PromotionWindow[], date: Date) =>
  useMemo(
    () => parsePromotionWindows(promotionWindows, { range: visibleRange(date) }),
    [promotionWindows, date]
  );
