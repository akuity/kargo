// @ts-nocheck cannot use class as "new Promotion" because they are being deprecated in new protobuf version, using those would mean tech debt

import { describe, expect, test } from 'vitest';

import { objectToYAML, promotionCompareFn } from './promotion';

test('promotionCompareFn', () => {
  expect(
    promotionCompareFn(
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') } }
      },
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:02') } }
      }
    )
  ).toBe(1);

  expect(
    promotionCompareFn(
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:02') } }
      },
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') } }
      }
    )
  ).toBe(-1);

  expect(
    promotionCompareFn(
      {
        metadata: {
          creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') },
          name: 'a'
        }
      },
      {
        metadata: {
          creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') },
          name: 'b'
        }
      }
    )
  ).toBe(-1);

  expect(
    promotionCompareFn(
      {
        metadata: {
          creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') },
          name: 'b'
        }
      },
      {
        metadata: {
          creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') },
          name: 'a'
        }
      }
    )
  ).toBe(1);

  expect(
    [
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') } },
        name: 10
      },
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:02') } },
        name: 1
      },
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:04') } },
        name: 2
      },
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:03') } },
        name: 100
      }
    ]
      .sort(promotionCompareFn)
      .map((p) => p.name)
  ).toStrictEqual([2, 100, 1, 10]);

  expect(
    [
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') } },
        name: 'a'
      },
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:01') } },
        name: 'b'
      },
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:04') } },
        name: 'c'
      },
      {
        metadata: { creationTimestamp: { toDate: () => new Date('01/01/2000 01:01:03') } },
        name: 'd'
      }
    ]
      .sort(promotionCompareFn)
      .map((p) => p.name)
  ).toStrictEqual(['c', 'd', 'a', 'b']);
});

describe('objectToYAML', () => {
  test('returns an empty string when there is no value', () => {
    expect(objectToYAML(undefined)).toBe('');
  });

  test('renders multi-line string values with real line breaks', () => {
    const output = {
      plan: 'line one\nline two\nline three'
    };

    const result = objectToYAML(output);

    // block scalar, not an escaped single line
    expect(result).not.toContain('\\n');
    expect(result).toContain('plan: |-');
    expect(result).toContain('  line one\n  line two\n  line three');
  });

  test('does not fold long single lines', () => {
    const longLine = 'a'.repeat(200);

    expect(objectToYAML({ value: longLine })).toBe(`value: ${longLine}\n`);
  });

  test('preserves scalar values', () => {
    expect(objectToYAML({ commit: 'abc123', count: 3 })).toBe('commit: abc123\ncount: 3\n');
  });
});
