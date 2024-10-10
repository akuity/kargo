import { expect, test } from 'vitest';

import { cleanEmptyObjectValues } from './helpers';

test('cleanEmptyObjectValues', () => {
  expect(cleanEmptyObjectValues({})).toStrictEqual({});

  expect(
    cleanEmptyObjectValues({
      a: [
        {
          b: {}
        }
      ]
    })
  ).toStrictEqual({});

  expect(
    cleanEmptyObjectValues({
      a: {
        b: {
          c: null
        },
        d: 'yes'
      },
      b: {
        c: [
          {
            d: null
          }
        ]
      }
    })
  ).toStrictEqual({
    a: {
      d: 'yes'
    }
  });

  expect(
    cleanEmptyObjectValues({
      a: {
        b: {
          c: null
        },
        d: undefined
      },
      b: {
        c: [
          {
            d: null
          }
        ]
      }
    })
  ).toStrictEqual({});

  expect(
    cleanEmptyObjectValues({
      a: {
        b: 'c',
        d: true,
        e: [
          {
            f: 'g'
          }
        ]
      }
    })
  ).toStrictEqual({
    a: {
      b: 'c',
      d: true,
      e: [
        {
          f: 'g'
        }
      ]
    }
  });
});
