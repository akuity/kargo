import { endOfMonth, endOfWeek, startOfMonth, startOfWeek } from 'date-fns';
import { useMemo } from 'react';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import { parsePromotionWindows } from './parse-promotion-windows';
import { CalendarView } from './promotion-calendar';

export const useGetPromotionWindowOccurrences = (
  promotionWindows: PromotionWindow[],
  view: CalendarView,
  date: Date
) =>
  useMemo(
    () =>
      parsePromotionWindows(promotionWindows, {
        range:
          view === 'dayGridMonth'
            ? { from: startOfWeek(startOfMonth(date)), to: endOfWeek(endOfMonth(date)) }
            : { from: startOfWeek(date), to: endOfWeek(date) }
      }),
    [promotionWindows, view, date]
  );
