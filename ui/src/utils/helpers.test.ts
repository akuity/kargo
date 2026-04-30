import { expect, test } from 'vitest';

import { cleanEmptyObjectValues, removePropertiesRecursively } from './helpers';

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

test('removePropertiesRecursively', () => {
  expect(
    removePropertiesRecursively(
      {
        a: {
          b: 'c'
        }
      },
      ['b']
    )
  ).toStrictEqual({ a: {} });

  expect(
    removePropertiesRecursively({ a: 'b', c: { d: { e: [{ f: 'g' }] } } }, ['f'])
  ).toStrictEqual({ a: 'b', c: { d: { e: [{}] } } });

  expect(
    removePropertiesRecursively({ a: 'b', c: { d: { e: [{ f: 'g' }, { h: 'i' }] } } }, ['f'])
  ).toStrictEqual({ a: 'b', c: { d: { e: [{}, { h: 'i' }] } } });
});
