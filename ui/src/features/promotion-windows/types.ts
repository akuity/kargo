import { PromotionWindow } from '@ui/gen/api/v2/models';

/**
 * A single materialized occurrence of a PromotionWindow.
 *
 * The API carries the schedule unparsed: `dtstart`/`dtend` as iCal date-times
 * and `rrule` as an RFC 5545 recurrence rule. Those three are dropped here and
 * replaced by the exact span they resolve to, so one recurring window expands
 * into many of these -- one per occurrence within the range asked for.
 */
export type PromotionWindowOccurrence = Omit<PromotionWindow, 'dtstart' | 'dtend' | 'rrule'> & {
  /**
   * rrule parses the dates and outputs them with a "0" offset, BUT IN REALITY
   * THESE ARE NOT UTC DATES -- the time is exactly what the user has in their
   * local time.
   */
  start: Date;
  end: Date;
};

export type PromotionWindowQuery =
  { before: Date; range?: never } | { before?: never; range: { from: Date; to: Date } };
