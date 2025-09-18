import { expect, test } from 'vitest';

import { getContrastTextColor } from './get-contrast-text-color';

test('getContrastTextColor', () => {
  expect(getContrastTextColor('')).toStrictEqual('white');
  expect(getContrastTextColor('black')).toStrictEqual('white');
  expect(getContrastTextColor('#ffffff')).toStrictEqual('black');
  expect(getContrastTextColor('#000000')).toStrictEqual('white');
  expect(getContrastTextColor('#ff0000')).toStrictEqual('white');
});
