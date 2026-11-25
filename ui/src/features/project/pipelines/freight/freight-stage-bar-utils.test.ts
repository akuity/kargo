import { describe, expect, test } from 'vitest';

import { getStageBandColor, NEUTRAL_BAND_COLOR } from './freight-stage-bar-utils';

const stageColorMap = { test: '#ff0000' };

describe('getStageBandColor', () => {
  test("a Stage's band takes its color in the graph", () => {
    expect(getStageBandColor('test', stageColorMap, true)).toBe('#ff0000');
  });

  test('every band is neutral when Stage colors are off', () => {
    expect(getStageBandColor('test', stageColorMap, false)).toBe(NEUTRAL_BAND_COLOR);
  });

  test('a Stage with no color gets the neutral band', () => {
    expect(getStageBandColor('prod', stageColorMap, true)).toBe(NEUTRAL_BAND_COLOR);
  });
});
