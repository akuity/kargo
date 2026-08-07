import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { Freight, Stage } from '@ui/gen/api/v2/models';

import { getSoakTime } from './get-soak-time';

const makeFreight = (stageName: string, since: string): Freight =>
  ({
    status: {
      currentlyIn: {
        [stageName]: { since }
      }
    }
  }) as unknown as Freight;

const makeStage = (name: string): Stage => ({ metadata: { name } }) as unknown as Stage;

describe('getSoakTime', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test('returns empty string when requiredSoakTime is empty', () => {
    expect(
      getSoakTime({
        freight: {} as Freight,
        freightInStage: makeStage('stage-a'),
        requiredSoakTime: ''
      })
    ).toBe('');
  });

  test('returns empty string when freight is not currently in the source stage', () => {
    expect(
      getSoakTime({
        freight: { status: { currentlyIn: {} } } as unknown as Freight,
        freightInStage: makeStage('stage-a'),
        requiredSoakTime: '1h'
      })
    ).toBe('');
  });

  test('returns null when freight has already soaked', () => {
    vi.setSystemTime(new Date('2024-01-01T12:00:00Z'));

    expect(
      getSoakTime({
        freight: makeFreight('stage-a', '2024-01-01T10:00:00Z'),
        freightInStage: makeStage('stage-a'),
        requiredSoakTime: '1h'
      })
    ).toBeNull();
  });

  test('returns remaining duration when freight has not yet soaked', () => {
    vi.setSystemTime(new Date('2024-01-01T12:00:00Z'));

    const result = getSoakTime({
      freight: makeFreight('stage-a', '2024-01-01T11:30:00Z'),
      freightInStage: makeStage('stage-a'),
      requiredSoakTime: '1h'
    });

    expect(result).toMatchObject({ minutes: 30 });
  });

  test.each([
    { duration: '1h10m', expected: { hours: 1, minutes: 10 } },
    { duration: '1h30m', expected: { hours: 1, minutes: 30 } },
    { duration: '2h5m', expected: { hours: 2, minutes: 5 } },
    { duration: '45m30s', expected: { minutes: 45, seconds: 30 } },
    { duration: '1h30m10s', expected: { hours: 1, minutes: 30, seconds: 10 } },
    { duration: '2h5m30s', expected: { hours: 2, minutes: 5, seconds: 30 } }
  ])('parses multi-part duration $duration', ({ duration, expected }) => {
    vi.setSystemTime(new Date('2024-01-01T12:00:00Z'));

    const result = getSoakTime({
      freight: makeFreight('stage-a', '2024-01-01T12:00:00Z'),
      freightInStage: makeStage('stage-a'),
      requiredSoakTime: duration
    });

    expect(result).toMatchObject(expected);
  });

  test('parses seconds-only duration (30s)', () => {
    vi.setSystemTime(new Date('2024-01-01T12:00:00Z'));

    const result = getSoakTime({
      freight: makeFreight('stage-a', '2024-01-01T12:00:00Z'),
      freightInStage: makeStage('stage-a'),
      requiredSoakTime: '30s'
    });

    expect(result).toMatchObject({ seconds: 30 });
  });

  test('parses minutes-only duration (10m)', () => {
    vi.setSystemTime(new Date('2024-01-01T12:00:00Z'));

    const result = getSoakTime({
      freight: makeFreight('stage-a', '2024-01-01T12:00:00Z'),
      freightInStage: makeStage('stage-a'),
      requiredSoakTime: '10m'
    });

    expect(result).toMatchObject({ minutes: 10 });
  });

  test('returns null exactly when soak time is met', () => {
    vi.setSystemTime(new Date('2024-01-01T13:00:00Z'));

    expect(
      getSoakTime({
        freight: makeFreight('stage-a', '2024-01-01T12:00:00Z'),
        freightInStage: makeStage('stage-a'),
        requiredSoakTime: '1h'
      })
    ).toBeNull();
  });
});
