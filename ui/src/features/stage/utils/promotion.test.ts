// @ts-nocheck cannot use class as "new Promotion" because they are being deprecated in new protobuf version, using those would mean tech debt

import { describe, expect, test } from 'vitest';

import { objectToYAML, promotionCompareFn } from './promotion';

test('promotionCompareFn', () => {
  expect(
    promotionCompareFn(
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:01Z' }
      },
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:02Z' }
      }
    )
  ).toBe(1);

  expect(
    promotionCompareFn(
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:02Z' }
      },
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:01Z' }
      }
    )
  ).toBe(-1);

  expect(
    promotionCompareFn(
      {
        metadata: {
          creationTimestamp: '2000-01-01T01:01:01Z',
          name: 'a'
        }
      },
      {
        metadata: {
          creationTimestamp: '2000-01-01T01:01:01Z',
          name: 'b'
        }
      }
    )
  ).toBe(-1);

  expect(
    promotionCompareFn(
      {
        metadata: {
          creationTimestamp: '2000-01-01T01:01:01Z',
          name: 'b'
        }
      },
      {
        metadata: {
          creationTimestamp: '2000-01-01T01:01:01Z',
          name: 'a'
        }
      }
    )
  ).toBe(1);

  expect(
    [
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:01Z' },
        name: 10
      },
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:02Z' },
        name: 1
      },
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:04Z' },
        name: 2
      },
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:03Z' },
        name: 100
      }
    ]
      .sort(promotionCompareFn)
      .map((p) => p.name)
  ).toStrictEqual([2, 100, 1, 10]);

  expect(
    [
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:01Z' },
        name: 'a'
      },
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:01Z' },
        name: 'b'
      },
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:04Z' },
        name: 'c'
      },
      {
        metadata: { creationTimestamp: '2000-01-01T01:01:03Z' },
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
