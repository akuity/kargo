import { describe, expect, test } from 'vitest';

import { defaultValueFormatter } from './utils';

describe('defaultValueFormatter', () => {
  const testCases: {
    name: string;
    value?: number | string | null;
    expected: string;
  }[] = [
    // Regression: a conditionKey present in the deduped union but absent from a
    // data point's chartValue yields undefined. Guarding only null previously
    // threw "Cannot read properties of undefined (reading 'toString')".
    { name: 'undefined coalesces to empty string', value: undefined, expected: '' },
    { name: 'null coalesces to empty string', value: null, expected: '' },
    { name: 'zero is stringified, not treated as empty', value: 0, expected: '0' },
    { name: 'number is stringified', value: 4.05, expected: '4.05' },
    { name: 'empty string is preserved', value: '', expected: '' },
    { name: 'string is preserved', value: 'n/a', expected: 'n/a' }
  ];

  for (const testCase of testCases) {
    test(testCase.name, () => {
      expect(defaultValueFormatter(testCase.value)).toBe(testCase.expected);
    });
  }
});
