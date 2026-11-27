// How long the text takes to slide each pixel of its hidden part into view,
// so long and short overflows move at the same pace.
const SLIDE_MS_PER_PX = 20;
const MIN_SLIDE_MS = 300;

// getOverflow is how much of a text is cut off by its box, in whole pixels.
// Sub-pixel differences are rounding, not overflow, and count as none.
export const getOverflow = (textWidth: number, boxWidth: number): number => {
  const hidden = textWidth - boxWidth;

  return hidden > 0.5 ? Math.ceil(hidden) : 0;
};

// getSlideDuration is how long, in milliseconds, overflowing text takes to
// slide its hidden part into view.
export const getSlideDuration = (overflow: number): number =>
  Math.max(MIN_SLIDE_MS, overflow * SLIDE_MS_PER_PX);
