// @ts-nocheck cannot use class as "new Promotion" because they are being deprecated in new protobuf version, using those would mean tech debt

import { expect, test } from 'vitest';

import { promotionCompareFn } from './promotion';

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
