import { useCallback, useEffect, useRef, useState } from 'react';

import { FreightTimelineControllerContextType } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';

const PAGE_SIZE = 20;
const MIN_CARD_WIDTH = 140;
const CARD_GAP = 4;

type PreferredFilter = FreightTimelineControllerContextType['preferredFilter'];

export type FreightCarousel = {
  viewportRef: React.RefObject<HTMLDivElement | null>;
  stripRef: React.RefObject<HTMLDivElement | null>;
  cardWidth: number;
  offset: number;
  visibleCount: number;
  loadMore: () => void;
  slideLeft: () => void;
  slideRight: () => void;
  canSlideLeft: boolean;
  canSlideRight: boolean;
};

/**
 * Drives the freight strip's horizontal pagination: computes per-card width
 * from the viewport, tracks scroll offset, and lazily reveals more items as
 * the user pages right. Resets visible count and offset whenever the filter
 * inputs that affect the displayed set change.
 */
export const useFreightCarousel = (
  itemCount: number,
  preferredFilter: PreferredFilter
): FreightCarousel => {
  const viewportRef = useRef<HTMLDivElement>(null);
  const stripRef = useRef<HTMLDivElement>(null);
  // Cards-per-viewport, derived from width. Tracked in a ref so the
  // filter-reset effect can size visibleCount to the viewport without
  // depending on resize-effect state.
  const cardsPerPageRef = useRef(1);

  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);
  const [offset, setOffset] = useState(0);
  const [cardWidth, setCardWidth] = useState(MIN_CARD_WIDTH);
  const [canSlideRight, setCanSlideRight] = useState(false);

  useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;

    const compute = () => {
      // Use sub-pixel viewport width and keep exact card width — flooring
      // discards fractional pixels, and the accumulated loss across N cards
      // makes the next card peek through on certain resolutions.
      const W = viewport.getBoundingClientRect().width;
      if (W <= 0) return;
      const n = Math.max(1, Math.floor((W + CARD_GAP) / (MIN_CARD_WIDTH + CARD_GAP)));
      const exactW = (W - (n - 1) * CARD_GAP) / n;
      cardsPerPageRef.current = n;
      setCardWidth(exactW);
      setOffset(0);
      // Ensure the strip fills the viewport (n cards) and keeps a page of
      // buffer ready for the first slide-right. Never shrink — the user may
      // have already paged past this point.
      setVisibleCount((prev) => Math.max(prev, n * 2));
    };

    compute();
    const ro = new ResizeObserver(compute);
    ro.observe(viewport);
    return () => ro.disconnect();
  }, []);

  useEffect(() => {
    setVisibleCount(Math.max(PAGE_SIZE, cardsPerPageRef.current * 2));
    setOffset(0);
  }, [
    preferredFilter.sources,
    preferredFilter.timerange,
    preferredFilter.warehouses,
    preferredFilter.hideUnusedFreights
  ]);

  const loadMore = useCallback(() => {
    setVisibleCount((prev) => Math.min(prev + PAGE_SIZE, itemCount));
  }, [itemCount]);

  const slideLeft = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    // With exact card widths, a page is W + gap (the trailing gap after the
    // last card on the page); using just W would drift by `gap` per slide.
    const stride = viewport.getBoundingClientRect().width + CARD_GAP;
    setOffset((prev) => Math.max(0, prev - stride));
  }, []);

  const slideRight = useCallback(() => {
    const viewport = viewportRef.current;
    const strip = stripRef.current;
    if (!viewport || !strip) return;

    const W = viewport.getBoundingClientRect().width;
    const stride = W + CARD_GAP;
    const maxOffset = Math.max(0, strip.scrollWidth - W);
    const hasMore = visibleCount < itemCount;

    setOffset((prev) => {
      const next = prev + stride;
      if (next > maxOffset) {
        if (hasMore) {
          loadMore();
          return next;
        }
        return maxOffset;
      }
      return next;
    });
  }, [itemCount, loadMore, visibleCount]);

  useEffect(() => {
    const viewport = viewportRef.current;
    const strip = stripRef.current;
    if (!viewport || !strip) {
      setCanSlideRight(false);
      return;
    }
    const maxOffset = Math.max(0, strip.scrollWidth - viewport.clientWidth);
    setCanSlideRight(offset < maxOffset - 1 || visibleCount < itemCount);
    // cardWidth tracks viewport size changes from the ResizeObserver. Without
    // it, a resize that leaves offset at 0 would not re-run this effect, so
    // canSlideRight could stay stale when the strip starts/stops overflowing.
  }, [offset, visibleCount, itemCount, cardWidth]);

  return {
    viewportRef,
    stripRef,
    cardWidth,
    offset,
    visibleCount,
    loadMore,
    slideLeft,
    slideRight,
    canSlideLeft: offset > 0,
    canSlideRight
  };
};
