import { RRule } from 'rrule';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import { PromotionWindowOccurrence, PromotionWindowQuery } from './types';

// example output values
// DTSTART:<raw> or DTSTART;<raw>
// DTSTART;TZID=America/Denver:20181101T190000
// DTSTART:20120201T023000Z
export const dtstartLiteral = (raw?: string) =>
  `DTSTART${raw?.startsWith('TZID=') ? ';' : ':'}${raw}`;

export const parsePromotionWindows = (
  promotionWindows: PromotionWindow[],
  query: PromotionWindowQuery
): PromotionWindowOccurrence[] => {
  const windows: PromotionWindowOccurrence[] = [];

  for (const promotionWindow of promotionWindows) {
    const { dtstart, dtend, rrule, ...rest } = promotionWindow;

    /*
     * these dates are 0 offset BUT IN LOCAL TIMEZONE
     */
    const start = RRule.fromString(dtstartLiteral(dtstart)).options.dtstart;
    const end = RRule.fromString(dtstartLiteral(dtend)).options.dtstart;

    const duration = end.getTime() - start.getTime();
    let occurrences: Date[] = [];

    if (!rrule && query.before && start <= query.before) {
      windows.push({ ...rest, start, end });
    }

    if (!rrule && query.range && end > query.range.from && start < query.range.to) {
      windows.push({ ...rest, start, end });
    }

    if (rrule) {
      try {
        const rule = RRule.fromString(`${dtstartLiteral(dtstart)}\nRRULE:${rrule}`);

        if (query.before) {
          const occurrence = rule.before(query.before, true);
          occurrences = occurrence ? [occurrence] : [];
        }

        if (query.range) {
          occurrences = rule.between(
            new Date(query.range.from.getTime() - duration),
            query.range.to
          );
        }
      } catch {
        occurrences = [];
      }

      for (const occurrence of occurrences) {
        windows.push({
          ...rest,
          start: occurrence,
          end: new Date(occurrence.getTime() + duration)
        });
      }
    }
  }

  return windows;
};
