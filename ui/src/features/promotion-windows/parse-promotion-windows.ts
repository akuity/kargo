import { tzOffset } from '@date-fns/tz';
import { RRule } from 'rrule';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import { PromotionWindowOccurrence, PromotionWindowQuery } from './types';
import { browserTimeZone, MINUTE_MS } from './viewer-clock';

// example output values
// DTSTART:<raw> or DTSTART;<raw>
// DTSTART;TZID=America/Denver:20181101T190000
// DTSTART:20120201T023000Z
export const dtstartLiteral = (raw?: string) =>
  `DTSTART${raw?.startsWith('TZID=') ? ';' : ':'}${raw}`;

const viewerClockDate = (raw?: string) => {
  const { dtstart, tzid } = RRule.fromString(dtstartLiteral(raw)).options;

  if (!tzid || tzid.toUpperCase() === 'UTC') {
    return dtstart;
  }

  const windowOffset = tzOffset(tzid, dtstart);
  const viewerOffset = tzOffset(browserTimeZone(), dtstart);

  return new Date(dtstart.getTime() - (windowOffset - viewerOffset) * MINUTE_MS);
};

export const parsePromotionWindows = (
  promotionWindows: PromotionWindow[],
  query: PromotionWindowQuery
): PromotionWindowOccurrence[] => {
  const windows: PromotionWindowOccurrence[] = [];

  for (const promotionWindow of promotionWindows) {
    const { dtstart, dtend, rrule, ...rest } = promotionWindow;

    if (!dtstart || !dtend) {
      continue;
    }

    /*
     * these dates are 0 offset BUT IN LOCAL TIMEZONE
     */
    let start: Date;
    let end: Date;
    try {
      /**
       * if you raw parse dtstart line, you will get time in timezone of TZID
       * this behaviour is inconsistent with rrule.all(), where the returned dates are in your
       * browser timezone + 0 offset
       * so how do we solve: we first get the offset from UTC of TZ, then we get the offset from UTC of your browser timezone
       * then we come up with final offset of your browser and TZID and adjust time accordingly
       * so final output is closer to our mental model
       */
      start = viewerClockDate(dtstart);
      end = viewerClockDate(dtend);
    } catch {
      continue;
    }

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
