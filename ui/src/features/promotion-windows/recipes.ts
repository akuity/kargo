import {
  addDays,
  addHours,
  addMinutes,
  addMonths,
  addWeeks,
  addYears,
  endOfMonth,
  isBefore,
  setDay,
  startOfDay,
  startOfMonth
} from 'date-fns';

import { PromotionWindow, PromotionWindowKind } from '@ui/gen/api/v2/models';

import { promotionWindowFromRange } from './promotion-window-form';

const WEEKDAY_RRULE = 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR';

export type PromotionWindowRecipe = {
  key: string;
  label: string;
  description: string;
  kind: PromotionWindowKind;
  create: () => PromotionWindow;
};

export const promotionWindowRecipes: PromotionWindowRecipe[] = [
  {
    key: 'business-hours',
    label: 'Business hours',
    description:
      'Promotions only on weekdays, 09:00-17:00. An Allow window inverts the default, so every other hour is shut.',
    kind: PromotionWindowKind.PromotionWindowKindAllow,
    create: () => {
      const midnight = startOfDay(new Date());

      return {
        ...promotionWindowFromRange(addHours(midnight, 9), addHours(midnight, 17)),
        name: 'business-hours',
        kind: PromotionWindowKind.PromotionWindowKindAllow,
        rrule: WEEKDAY_RRULE
      };
    }
  },
  {
    key: 'weekend-freeze',
    label: 'Weekend freeze',
    description:
      'Shuts promotions from Saturday 00:00 for 48 hours, every week. Everything else stays open.',
    kind: PromotionWindowKind.PromotionWindowKindDeny,
    create: () => {
      const saturday = startOfDay(setDay(new Date(), 6));

      return {
        ...promotionWindowFromRange(saturday, addHours(saturday, 48)),
        name: 'weekend-freeze',
        kind: PromotionWindowKind.PromotionWindowKindDeny,
        rrule: 'FREQ=WEEKLY;BYDAY=SA'
      };
    }
  },
  {
    key: 'one-shot-freeze',
    label: 'One-shot freeze',
    description:
      'A single freeze with a hard end and no recurrence - a peak-traffic week, a conference, an audit.',
    kind: PromotionWindowKind.PromotionWindowKindDeny,
    create: () => {
      const midnight = startOfDay(new Date());

      return {
        ...promotionWindowFromRange(midnight, addDays(midnight, 5)),
        name: 'one-shot-freeze',
        kind: PromotionWindowKind.PromotionWindowKindDeny
      };
    }
  },
  {
    key: 'incident-hold',
    label: 'Indefinite hold',
    description:
      'Freezes promotions from now on. There is no "forever" token, so this picks a far-future end you can shorten once the incident closes.',
    kind: PromotionWindowKind.PromotionWindowKindDeny,
    create: () => {
      const midnight = startOfDay(new Date());

      return {
        ...promotionWindowFromRange(midnight, addYears(midnight, 10)),
        name: 'incident-hold',
        kind: PromotionWindowKind.PromotionWindowKindDeny
      };
    }
  },
  {
    key: 'release-train',
    label: 'Release train',
    description: 'Promotions only during a monthly slot: the first Tuesday, 10:00-12:00.',
    kind: PromotionWindowKind.PromotionWindowKindAllow,
    create: () => {
      const firstOfNextMonth = startOfMonth(addMonths(new Date(), 1));
      const tuesdayThatWeek = setDay(firstOfNextMonth, 2);
      const firstTuesday = isBefore(tuesdayThatWeek, firstOfNextMonth)
        ? addWeeks(tuesdayThatWeek, 1)
        : tuesdayThatWeek;

      return {
        ...promotionWindowFromRange(addHours(firstTuesday, 10), addHours(firstTuesday, 12)),
        name: 'release-train',
        kind: PromotionWindowKind.PromotionWindowKindAllow,
        rrule: 'FREQ=MONTHLY;BYDAY=+1TU'
      };
    }
  },
  {
    key: 'month-end-close',
    label: 'Month-end close',
    description: 'Shuts the last calendar day of every month, for the books to close.',
    kind: PromotionWindowKind.PromotionWindowKindDeny,
    create: () => {
      const lastDay = startOfDay(endOfMonth(new Date()));

      return {
        ...promotionWindowFromRange(lastDay, addDays(lastDay, 1)),
        name: 'month-end-close',
        kind: PromotionWindowKind.PromotionWindowKindDeny,
        rrule: 'FREQ=MONTHLY;BYMONTHDAY=-1'
      };
    }
  },
  {
    key: 'prod-guard',
    label: 'Production guard',
    description:
      'Business hours, but only for Stages labelled env=prod. Everything else is untouched.',
    kind: PromotionWindowKind.PromotionWindowKindAllow,
    create: () => {
      const midnight = startOfDay(new Date());

      return {
        ...promotionWindowFromRange(addHours(midnight, 9), addHours(midnight, 17)),
        name: 'prod-guard',
        kind: PromotionWindowKind.PromotionWindowKindAllow,
        rrule: WEEKDAY_RRULE,
        stageSelector: { matchLabels: { env: 'prod' } }
      };
    }
  },
  {
    key: 'carve-out',
    label: 'Carve-out',
    description:
      'A short Deny laid over an existing Allow window - weekdays 10:00-10:15. Layering is the only way to express an exception; there is no EXDATE.',
    kind: PromotionWindowKind.PromotionWindowKindDeny,
    create: () => {
      const midnight = startOfDay(new Date());

      return {
        ...promotionWindowFromRange(addHours(midnight, 10), addMinutes(addHours(midnight, 10), 15)),
        name: 'carve-out',
        kind: PromotionWindowKind.PromotionWindowKindDeny,
        rrule: WEEKDAY_RRULE
      };
    }
  }
];
