import { endOfMonth, endOfWeek, startOfMonth, startOfWeek } from 'date-fns';
import { useMemo } from 'react';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import { parsePromotionWindows } from './parse-promotion-windows';
import type { CalendarView } from './promotion-calendar';
import { toViewerClockDate } from './viewer-clock';

export const visibleRange = (view: CalendarView, date: Date) => {
  const [from, to] =
    view === 'dayGridMonth'
      ? [startOfWeek(startOfMonth(date)), endOfWeek(endOfMonth(date))]
      : [startOfWeek(date), endOfWeek(date)];

  // IMPORTANT - BEFORE YOU GIVE RRULE TO QUERY, YOU NEED TO STAMP THE TIMEZONE OFFSET IT IS IN
  return { from: toViewerClockDate(from), to: toViewerClockDate(to) };
};

export const useGetPromotionWindowOccurrences = (
  promotionWindows: PromotionWindow[],
  view: CalendarView,
  date: Date
) =>
  useMemo(
    () => parsePromotionWindows(promotionWindows, { range: visibleRange(view, date) }),
    [promotionWindows, view, date]
  );
