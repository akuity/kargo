import { describe, expect, it } from 'vitest';

import { PromotionWindow } from '@ui/gen/api/v2/models';

import { isPromotionWindowOpen } from './is-promotion-window-open';

const withTimeZone = <T>(timeZone: string, run: () => T): T => {
  const original = process.env.TZ;
  process.env.TZ = timeZone;
  try {
    return run();
  } finally {
    process.env.TZ = original;
  }
};

const allow = (overrides: Partial<PromotionWindow> = {}): PromotionWindow => ({
  name: 'allow-w',
  kind: 'Allow',
  dtstart: '20260824T090000Z',
  dtend: '20260824T170000Z',
  ...overrides
});

const deny = (overrides: Partial<PromotionWindow> = {}): PromotionWindow => ({
  name: 'deny-w',
  kind: 'Deny',
  dtstart: '20260824T100000Z',
  dtend: '20260824T101500Z',
  ...overrides
});

const openAt = (windows: PromotionWindow[], iso: string) =>
  withTimeZone('UTC', () => isPromotionWindowOpen(windows, Date.parse(iso)));

describe('isPromotionWindowOpen', () => {
  it('is open when no windows are configured', () => {
    expect(openAt([], '2026-08-24T12:00:00Z')).toBe(true);
  });

  it('is open when the only window is an idle Deny', () => {
    expect(openAt([deny()], '2026-08-24T12:00:00Z')).toBe(true);
  });

  it('is closed while a Deny is active', () => {
    expect(openAt([deny()], '2026-08-24T10:05:00Z')).toBe(false);
  });

  it('is open inside an Allow window', () => {
    expect(openAt([allow()], '2026-08-24T12:00:00Z')).toBe(true);
  });

  it('is closed outside an Allow window', () => {
    expect(openAt([allow()], '2026-08-24T20:00:00Z')).toBe(false);
  });

  it('lets an active Deny beat an active Allow', () => {
    expect(openAt([allow(), deny()], '2026-08-24T10:05:00Z')).toBe(false);
  });

  it('is open inside an Allow window while a Deny is idle', () => {
    expect(openAt([allow(), deny()], '2026-08-24T12:00:00Z')).toBe(true);
  });

  it('treats the start of a window as inside it', () => {
    expect(openAt([deny()], '2026-08-24T10:00:00Z')).toBe(false);
  });

  it('treats the end of a window as outside it', () => {
    expect(openAt([deny()], '2026-08-24T10:15:00Z')).toBe(true);
  });

  it('is closed when an Allow exists but only matches another time', () => {
    expect(openAt([allow({ rrule: 'FREQ=WEEKLY;BYDAY=MO' })], '2026-08-25T12:00:00Z')).toBe(false);
  });

  it('is open inside a recurring Allow occurrence', () => {
    expect(
      openAt([allow({ rrule: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR' })], '2026-08-25T12:00:00Z')
    ).toBe(true);
  });

  it('is closed inside a recurring Deny carve-out', () => {
    expect(
      openAt(
        [
          allow({ rrule: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR' }),
          deny({ rrule: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR' })
        ],
        '2026-08-25T10:05:00Z'
      )
    ).toBe(false);
  });

  it('ignores a disabled Deny that would otherwise be active', () => {
    expect(openAt([deny({ disabled: true })], '2026-08-24T10:05:00Z')).toBe(true);
  });

  it('ignores a disabled Allow when looking for an explicit Allow', () => {
    expect(openAt([allow({ disabled: true })], '2026-08-24T20:00:00Z')).toBe(true);
  });

  it('keeps evaluating the enabled windows alongside a disabled one', () => {
    expect(openAt([allow(), deny({ disabled: true })], '2026-08-24T10:05:00Z')).toBe(true);
  });

  it('is closed inside a long-running one-shot freeze', () => {
    expect(
      openAt(
        [deny({ dtstart: '20260825T000000Z', dtend: '20360825T000000Z' })],
        '2027-01-01T00:00:00Z'
      )
    ).toBe(false);
  });

  it('is closed part-way through a 48h recurring freeze', () => {
    expect(
      openAt(
        [
          deny({
            dtstart: '20260829T000000Z',
            dtend: '20260831T000000Z',
            rrule: 'FREQ=WEEKLY'
          })
        ],
        '2026-08-30T12:00:00Z'
      )
    ).toBe(false);
  });
});
