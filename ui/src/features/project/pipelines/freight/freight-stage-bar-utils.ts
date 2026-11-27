import type { ColorMap } from '@ui/features/stage/utils';

// Used for every band when Stage colors are switched off in the graph, so the
// bar does not show colors the graph no longer does.
export const NEUTRAL_BAND_COLOR = 'var(--kargo-color-text-quaternary)';

// getStageBandColor is the color of a Stage's band in a freight card's Stage
// bar: the Stage's color in the graph, or a neutral one when Stage colors are
// off or the Stage has none.
export const getStageBandColor = (
  stageName: string,
  stageColorMap: ColorMap,
  showColors: boolean
): string => (showColors && stageColorMap[stageName]) || NEUTRAL_BAND_COLOR;
