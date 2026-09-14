import { afterEach, describe, expect, it, vi } from 'vitest';

import { promotionWindowRecipes } from './recipes';

const HOUR_MS = 3_600_000;

const parseICal = (literal: string) => {
  const raw = literal.slice(literal.indexOf(':') + 1);
  return new Date(
    Number(raw.slice(0, 4)),
    Number(raw.slice(4, 6)) - 1,
    Number(raw.slice(6, 8)),
    Number(raw.slice(9, 11)),
    Number(raw.slice(11, 13))
  );
};

const create = (key: string) => {
  const window = promotionWindowRecipes.find((recipe) => recipe.key === key)!.create();
  return { window, start: parseICal(window.dtstart!), end: parseICal(window.dtend!) };
};

afterEach(() => vi.useRealTimers());

describe('promotionWindowRecipes', () => {
  it('names every window after its own key', () => {
    for (const recipe of promotionWindowRecipes) {
      const window = recipe.create();
      expect(window.name).toBe(recipe.key);
      expect(window.kind).toBe(recipe.kind);
    }
  });

  it('ends every window after it starts', () => {
    for (const recipe of promotionWindowRecipes) {
      const window = recipe.create();
      expect(parseICal(window.dtend!).getTime()).toBeGreaterThan(
        parseICal(window.dtstart!).getTime()
      );
    }
  });

  it('scopes the production guard to env=prod Stages', () => {
    expect(create('prod-guard').window.stageSelector).toEqual({ matchLabels: { env: 'prod' } });
  });

  it('lays the carve-out inside business hours', () => {
    const { start, end } = create('carve-out');
    expect([start.getHours(), start.getMinutes()]).toEqual([10, 0]);
    expect([end.getHours(), end.getMinutes()]).toEqual([10, 15]);
  });

  it('runs business hours 09:00-17:00 on weekdays', () => {
    const { window, start, end } = create('business-hours');
    expect(start.getHours()).toBe(9);
    expect(end.getHours()).toBe(17);
    expect(window.rrule).toBe('FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR');
  });
});

// The recurring recipes are anchored on "now", so pin the clock across month,
// year and DST boundaries where the week and month arithmetic is most fragile.
describe.each([
  '2026-01-01T00:00:00',
  '2026-01-31T23:30:00',
  '2026-02-28T12:00:00',
  '2026-03-08T03:30:00',
  '2026-05-31T12:00:00',
  '2026-08-30T00:00:00',
  '2026-11-01T00:00:00',
  '2026-12-31T18:00:00',
  '2027-02-27T09:00:00'
])('promotionWindowRecipes anchored at %s', (now) => {
  const reference = () => new Date(now);
  const freeze = () => vi.useFakeTimers({ now: reference(), toFake: ['Date'] });

  it('starts the weekend freeze on Saturday midnight for 48 hours', () => {
    freeze();
    const { start, end } = create('weekend-freeze');

    expect(start.getDay()).toBe(6);
    expect([start.getHours(), start.getMinutes()]).toEqual([0, 0]);
    expect(end.getTime() - start.getTime()).toBe(48 * HOUR_MS);
  });

  it('puts the release train on the first Tuesday of next month, 10:00-12:00', () => {
    freeze();
    const { start, end } = create('release-train');
    const nextMonth = new Date(reference().getFullYear(), reference().getMonth() + 1, 1);

    expect(start.getDay()).toBe(2);
    expect(start.getDate()).toBeLessThanOrEqual(7);
    expect(start.getMonth()).toBe(nextMonth.getMonth());
    expect(start.getFullYear()).toBe(nextMonth.getFullYear());
    expect(start.getHours()).toBe(10);
    expect(end.getHours()).toBe(12);
  });

  it('covers the final calendar day for the month-end close', () => {
    freeze();
    const { start, end } = create('month-end-close');
    const lastDay = new Date(reference().getFullYear(), reference().getMonth() + 1, 0).getDate();

    expect(start.getDate()).toBe(lastDay);
    expect(end.getTime() - start.getTime()).toBe(24 * HOUR_MS);
  });

  it('ends the indefinite hold ten years out', () => {
    freeze();
    const { start, end } = create('incident-hold');

    expect(end.getFullYear() - start.getFullYear()).toBe(10);
  });

  it('runs the one-shot freeze for five days from midnight', () => {
    freeze();
    const { window, start, end } = create('one-shot-freeze');

    expect([start.getHours(), start.getMinutes()]).toEqual([0, 0]);
    expect(end.getTime() - start.getTime()).toBe(5 * 24 * HOUR_MS);
    expect(window.rrule).toBeUndefined();
  });
});
