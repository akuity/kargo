import { format } from 'date-fns';
import { describe, expect, it } from 'vitest';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import {
  ICAL_FORMAT,
  combine,
  formValuesFromPromotionWindow,
  promotionWindowFromFormValues,
  promotionWindowFromRange
} from './promotion-window-form-utils';

describe('ICAL_FORMAT', () => {
  it('renders an iCal local date-time', () => {
    expect(format(new Date(2026, 7, 24, 9, 5, 0), ICAL_FORMAT)).toBe('20260824T090500');
  });

  it('keeps a two-digit month and day', () => {
    expect(format(new Date(2026, 0, 2, 0, 0, 0), ICAL_FORMAT)).toBe('20260102T000000');
  });
});

describe('combine', () => {
  it('takes the calendar day from the first and the clock from the second', () => {
    expect(
      format(
        combine(new Date(2026, 7, 24, 1, 2, 3, 456), new Date(2020, 0, 1, 17, 30, 9, 9)),
        'yyyy-MM-dd HH:mm:ss.SSS'
      )
    ).toBe('2026-08-24 17:30:00.000');
  });
});

describe('promotionWindowFromRange', () => {
  it('writes both ends as zoned iCal literals', () => {
    const window = promotionWindowFromRange(
      new Date(2026, 0, 2, 9, 0),
      new Date(2026, 0, 2, 17, 0)
    );

    expect(window.dtstart).toMatch(/:20260102T090000$/);
    expect(window.dtend).toMatch(/:20260102T170000$/);
    expect(window.kind).toBe('Deny');
  });
});

describe('promotion window form round trip', () => {
  const cases: PromotionWindow[] = [
    {
      name: 'weekend-freeze',
      kind: 'Deny',
      dtstart: 'TZID=America/New_York:20260829T000000',
      dtend: 'TZID=America/New_York:20260831T000000',
      rrule: 'FREQ=WEEKLY;BYDAY=SA'
    },
    {
      name: 'business-hours',
      kind: 'Allow',
      description: 'weekdays only',
      dtstart: 'TZID=Asia/Tokyo:20260824T090000',
      dtend: 'TZID=Asia/Tokyo:20260824T170000',
      rrule: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR',
      stageSelector: { matchLabels: { env: 'prod' } }
    },
    {
      name: 'one-shot',
      kind: 'Deny',
      disabled: true,
      dtstart: 'TZID=UTC:20260101T000000',
      dtend: 'TZID=UTC:20260106T000000',
      stageSelector: { name: 'glob:prod-*' }
    },
    {
      name: 'critical-prod-freeze',
      kind: 'Deny',
      dtstart: 'TZID=UTC:20260101T000000',
      dtend: 'TZID=UTC:20260106T000000',
      stageSelector: { name: 'prod', matchLabels: { tier: 'critical' } }
    },
    {
      name: 'expression-selector',
      kind: 'Deny',
      dtstart: 'TZID=UTC:20260101T000000',
      dtend: 'TZID=UTC:20260106T000000',
      stageSelector: {
        matchExpressions: [{ key: 'tier', operator: 'In', values: ['critical', 'high'] }]
      }
    }
  ];

  it.each(cases)('survives form values and back for $name', (promotionWindow) => {
    expect(
      promotionWindowFromFormValues(formValuesFromPromotionWindow(promotionWindow), 'project')
    ).toEqual(promotionWindow);
  });

  it('keeps the label constraint when a name is also set', () => {
    const stageSelector = { name: 'prod', matchLabels: { tier: 'critical' } };
    const promotionWindow: PromotionWindow = {
      name: 'critical-prod-freeze',
      kind: 'Deny',
      dtstart: 'TZID=UTC:20260824T090000',
      dtend: 'TZID=UTC:20260824T170000',
      stageSelector
    };

    const saved = promotionWindowFromFormValues(
      formValuesFromPromotionWindow(promotionWindow),
      'project'
    );

    expect(saved.stageSelector).toEqual(stageSelector);
  });

  it('drops the selector entirely when neither constraint is set', () => {
    const promotionWindow: PromotionWindow = {
      name: 'everything',
      kind: 'Deny',
      dtstart: 'TZID=UTC:20260824T090000',
      dtend: 'TZID=UTC:20260824T170000'
    };

    const saved = promotionWindowFromFormValues(
      formValuesFromPromotionWindow(promotionWindow),
      'cluster'
    );

    expect(saved.stageSelector).toBeUndefined();
    expect(saved.projectSelector).toBeUndefined();
  });

  it('carries the cluster-scoped Project selector only in cluster scope', () => {
    const promotionWindow: PromotionWindow = {
      name: 'prod-guard',
      kind: 'Allow',
      dtstart: 'TZID=UTC:20260824T090000',
      dtend: 'TZID=UTC:20260824T170000',
      projectSelector: { name: 'regex:^prod-' }
    };

    const values = formValuesFromPromotionWindow(promotionWindow);

    expect(promotionWindowFromFormValues(values, 'cluster')).toEqual(promotionWindow);
    expect(promotionWindowFromFormValues(values, 'project').projectSelector).toBeUndefined();
  });
});
