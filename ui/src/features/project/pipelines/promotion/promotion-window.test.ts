import { describe, expect, test } from 'vitest';

import type { PromotionWindowStatus, Stage } from '@ui/gen/api/v2/models';

import {
  isPromotionWindowClosed,
  promotionWindowClosedMessage,
  promotionWindowClosedReason,
  promotionWindowNextOpen,
  promotionWindowReopensLabel
} from './promotion-window';

const stageWithWindow = (promotionWindowStatus?: PromotionWindowStatus): Stage =>
  ({
    status: { promotionWindowStatus }
  }) as Stage;

const now = new Date('2026-01-01T00:00:00Z');

describe('promotion window helpers', () => {
  test('isPromotionWindowClosed is false without a status', () => {
    expect(isPromotionWindowClosed()).toBe(false);
    expect(isPromotionWindowClosed({} as Stage)).toBe(false);
    expect(isPromotionWindowClosed(stageWithWindow())).toBe(false);
  });

  test('isPromotionWindowClosed follows the closed flag', () => {
    expect(isPromotionWindowClosed(stageWithWindow({ closed: false }))).toBe(false);
    expect(isPromotionWindowClosed(stageWithWindow({ closed: true }))).toBe(true);
  });

  test('promotionWindowClosedReason falls back to a default', () => {
    expect(promotionWindowClosedReason(stageWithWindow({ closed: true }))).toBe(
      'A promotion window currently forbids promotion of this Stage.'
    );
    expect(
      promotionWindowClosedReason(stageWithWindow({ closed: true, reason: 'Freeze "holiday".' }))
    ).toBe('Freeze "holiday".');
  });

  test('promotionWindowClosedReason terminates an unpunctuated reason', () => {
    const stage = stageWithWindow({
      closed: true,
      reason: 'No promotion window is currently active'
    });

    expect(promotionWindowClosedReason(stage)).toBe('No promotion window is currently active.');
    expect(promotionWindowClosedMessage(stage, now)).toBe(
      'No promotion window is currently active. No reopening time is known.'
    );
  });

  test('promotionWindowNextOpen parses only usable timestamps', () => {
    expect(promotionWindowNextOpen(stageWithWindow({ closed: true }))).toBeUndefined();
    expect(
      promotionWindowNextOpen(stageWithWindow({ closed: true, nextOpen: 'not-a-date' }))
    ).toBeUndefined();
    expect(
      promotionWindowNextOpen(stageWithWindow({ closed: true, nextOpen: '2026-01-01T02:00:00Z' }))
    ).toEqual(new Date('2026-01-01T02:00:00Z'));
  });

  test('promotionWindowClosedMessage is empty while promotion is permitted', () => {
    expect(promotionWindowClosedMessage(stageWithWindow({ closed: false }), now)).toBe('');
  });

  test('promotionWindowClosedMessage reports a missing reopening time', () => {
    expect(promotionWindowClosedMessage(stageWithWindow({ closed: true }), now)).toBe(
      'A promotion window currently forbids promotion of this Stage. No reopening time is known.'
    );
  });

  test('promotionWindowClosedMessage falls back to the default reason', () => {
    const stage = stageWithWindow({ closed: true, nextOpen: 'not-a-date' });

    expect(promotionWindowClosedMessage(stage, now)).toBe(
      'A promotion window currently forbids promotion of this Stage. No reopening time is known.'
    );
  });

  test('promotionWindowReopensLabel states when promotion resumes', () => {
    expect(promotionWindowReopensLabel(stageWithWindow({ closed: false }), now)).toBe('');
    expect(promotionWindowReopensLabel(stageWithWindow({ closed: true }), now)).toBe(
      'No known reopening time'
    );
    expect(
      promotionWindowReopensLabel(
        stageWithWindow({ closed: true, nextOpen: '2026-01-01T02:00:00Z' }),
        now
      )
    ).toBe('Reopens in about 2 hours');
  });

  test('promotionWindowClosedMessage includes the reason and reopening time', () => {
    const stage = stageWithWindow({
      closed: true,
      reason: 'Freeze "holiday" is active.',
      nextOpen: '2026-01-01T02:00:00Z'
    });

    expect(promotionWindowClosedMessage(stage, now)).toBe(
      'Freeze "holiday" is active. Promotion reopens in about 2 hours.'
    );
  });
});
