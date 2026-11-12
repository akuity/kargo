import { describe, expect, test } from 'vitest';

import { healthStatusToEnum, HealthStatus, healthTone } from './utils';

describe('healthStatusToEnum()', () => {
  test.each(['Healthy', 'Degraded', 'Progressing', 'Unhealthy', 'Unknown'])(
    'recognises %s',
    (status) => {
      expect(healthStatusToEnum(status)).toBe(status);
    }
  );

  test.each([undefined, '', 'Sideways'])('reports anything else as undefined (%s)', (status) => {
    expect(healthStatusToEnum(status)).toBe(HealthStatus.UNDEFINED);
  });
});

describe('healthTone()', () => {
  test.each([
    ['Healthy', 'success'],
    ['Degraded', 'error'],
    ['Unhealthy', 'error'],
    ['Progressing', 'processing'],
    ['Unknown', 'warning']
  ])('states %s in the %s tone', (status, tone) => {
    expect(healthTone(status)).toBe(tone);
  });

  // A status we do not know steps back rather than guessing at a verdict.
  test.each([undefined, '', 'Sideways'])('mutes an unrecognised status (%s)', (status) => {
    expect(healthTone(status)).toBe('muted');
  });
});
