import { addHours, format, getHours, getMinutes, isAfter, set, startOfHour } from 'date-fns';
import { Options, RRule } from 'rrule';
import { z } from 'zod';

import { dnsRegex } from '@ui/features/common/utils';
import { PromotionWindow, PromotionWindowKind } from '@ui/gen/api/v2/models';
import { zodValidators } from '@ui/utils/validators';

import { dtstartLiteral } from './parse-promotion-windows';
import {
  selectorFromValues,
  selectorSchema,
  selectorValues
} from './promotion-window-selector-utils';
import { fromViewerClockDate } from './viewer-clock';

export const ICAL_FORMAT = "yyyyMMdd'T'HHmmss";

export const promotionWindowFormSchema = z
  .object({
    name: zodValidators.requiredString
      .max(253)
      .regex(dnsRegex, 'Name must be a valid DNS subdomain.'),
    description: z.string().max(1024, 'Description must be 1024 characters or fewer.'),
    disabled: z.boolean(),
    kind: z.enum([
      PromotionWindowKind.PromotionWindowKindDeny,
      PromotionWindowKind.PromotionWindowKindAllow
    ]),
    timeZone: z.string().min(1),
    startDate: z.custom<Date>((value) => !!value, 'Required'),
    endDate: z.custom<Date>((value) => !!value, 'Required'),
    rrule: z.custom<Partial<Options> | null>(),
    stage: selectorSchema,
    project: selectorSchema
  })
  .refine((values) => isAfter(values.endDate, values.startDate), {
    message: 'End must be after start',
    path: ['endDate']
  });

export type PromotionWindowFormValues = z.infer<typeof promotionWindowFormSchema>;

export const combine = (date: Date, time: Date) =>
  set(date, { hours: getHours(time), minutes: getMinutes(time), seconds: 0, milliseconds: 0 });

export const browserTimeZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';

export const promotionWindowFromRange = (start: Date, end: Date): PromotionWindow => ({
  name: '',
  kind: PromotionWindowKind.PromotionWindowKindDeny,
  dtstart: `TZID=${browserTimeZone()}:${format(start, ICAL_FORMAT)}`,
  dtend: `TZID=${browserTimeZone()}:${format(end, ICAL_FORMAT)}`
});

export const formValuesFromPromotionWindow = (
  promotionWindow: PromotionWindow
): PromotionWindowFormValues => {
  const now = new Date();

  let timeZone = browserTimeZone();
  let startDate = now;
  let endDate = addHours(startOfHour(now), 1);
  let rrule: Partial<Options> | null = null;

  if (promotionWindow.dtstart && promotionWindow.dtend) {
    try {
      const start = RRule.fromString(dtstartLiteral(promotionWindow.dtstart)).options;
      const end = RRule.fromString(dtstartLiteral(promotionWindow.dtend)).options;

      timeZone = start.tzid ?? timeZone;
      startDate = fromViewerClockDate(start.dtstart);
      endDate = fromViewerClockDate(end.dtstart);
    } catch {
      startDate = now;
      endDate = addHours(startOfHour(now), 1);
    }
  }

  if (promotionWindow.rrule) {
    try {
      rrule = RRule.parseString(promotionWindow.rrule);
    } catch {
      rrule = null;
    }
  }

  return {
    name: promotionWindow.name,
    description: promotionWindow.description ?? '',
    disabled: promotionWindow.disabled ?? false,
    kind: promotionWindow.kind,
    timeZone,
    startDate,
    endDate,
    rrule,
    stage: selectorValues(promotionWindow.stageSelector),
    project: selectorValues(promotionWindow.projectSelector)
  };
};

export const promotionWindowFromFormValues = (
  values: PromotionWindowFormValues,
  scope: 'project' | 'cluster'
): PromotionWindow => {
  const rrule = values.rrule ? RRule.optionsToString(values.rrule).replace(/^RRULE:/, '') : '';
  const description = values.description.trim();
  const stageSelector = selectorFromValues(values.stage);
  const projectSelector = scope === 'cluster' ? selectorFromValues(values.project) : undefined;

  return {
    name: values.name.trim(),
    kind: values.kind,
    ...(description ? { description } : {}),
    ...(values.disabled ? { disabled: true } : {}),
    dtstart: `TZID=${values.timeZone}:${format(values.startDate, ICAL_FORMAT)}`,
    dtend: `TZID=${values.timeZone}:${format(values.endDate, ICAL_FORMAT)}`,
    ...(rrule ? { rrule } : {}),
    ...(stageSelector ? { stageSelector } : {}),
    ...(projectSelector ? { projectSelector } : {})
  };
};
