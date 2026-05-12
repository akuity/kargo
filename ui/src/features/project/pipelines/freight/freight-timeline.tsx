import { useDndContext } from '@dnd-kit/core';
import { faChevronLeft, faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';
import {
  CSSProperties,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState
} from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { useFreightTimelineControllerContext } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { timerangeToDate } from './filter-timerange-utils';
import { FreightCard } from './freight-card';
import { FreightExpandTile } from './freight-expand-tile';
import { PromotionModeHeader } from './promotion-mode-header';
import { filterFreightBySource, filterFreightByTimerange } from './source-catalogue-utils';
import { usePromotionEligibleFreight } from './use-promotion-eligible-freight';
import { useSoakTime } from './use-soak-time';

import './freight-timeline.less';

const PAGE_SIZE = 20;
const ARROW_BUTTON_WIDTH = 28;
const MIN_CARD_WIDTH = 140;
const CARD_GAP = 4;

export const FreightTimeline = (props: { freights: Freight[]; project: string }) => {
  const navigate = useNavigate();

  const colorContext = useContext(ColorContext);
  const freightTimelineControllerContext = useFreightTimelineControllerContext();
  const dictionaryContext = useDictionaryContext();
  const actionContext = useActionContext();

  const isPromotionMode =
    actionContext?.action?.type === IAction.PROMOTE ||
    actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM;

  const { promotionEligibleFreight, ...getPromotionEligibleFreightQuery } =
    usePromotionEligibleFreight(props.project);

  const soakTime = useSoakTime(props.freights);

  if (!freightTimelineControllerContext) {
    throw new Error('missing context freightTimelineControllerContext');
  }

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const filteredFreights: (Freight & {
    count?: number;
  })[] = useMemo(() => {
    let filtered = [...(props.freights || [])].sort((a, b) => {
      const t1 = timestampDate(a?.metadata?.creationTimestamp);

      const t2 = timestampDate(b?.metadata?.creationTimestamp);

      return (t2?.getTime() || 0) - (t1?.getTime() || 0);
    });

    filtered = filtered
      .map(filterFreightBySource(freightTimelineControllerContext.preferredFilter?.sources))
      .filter(Boolean) as Freight[];

    if (freightTimelineControllerContext.preferredFilter.timerange !== 'all-time') {
      filtered = filtered.filter(
        filterFreightByTimerange(
          timerangeToDate(freightTimelineControllerContext.preferredFilter.timerange)
        )
      );
    }

    if (freightTimelineControllerContext.preferredFilter.warehouses?.length > 0) {
      filtered = filtered.filter((f) =>
        freightTimelineControllerContext.preferredFilter.warehouses.includes(f.origin?.name || '')
      );
    }

    if (freightTimelineControllerContext.preferredFilter.hideUnusedFreights) {
      const newFiltered: (Freight & {
        count?: number;
      })[] = [];

      let count = 0;

      for (const f of filtered) {
        const inUse =
          (dictionaryContext?.freightInStages[f?.metadata?.name || '']?.length || 0) > 0;

        if (inUse) {
          if (count > 0) {
            newFiltered.push({
              ...f,
              count
            });
            count = 0;
          }

          newFiltered.push(f);
        } else {
          count++;
        }
      }

      filtered = [...newFiltered];
    }

    if (isPromotionMode) {
      filtered = filtered.filter((f) =>
        actionContext?.action?.stage?.spec?.requestedFreight?.find(
          (fr) => fr.origin?.name === f?.origin?.name
        )
      );
    }

    return filtered;
  }, [
    props.freights,
    props.freights.length,
    freightTimelineControllerContext.preferredFilter.sources,
    freightTimelineControllerContext.preferredFilter.timerange,
    freightTimelineControllerContext.preferredFilter.warehouses,
    freightTimelineControllerContext.preferredFilter.hideUnusedFreights,
    dictionaryContext?.freightInStages,
    isPromotionMode,
    actionContext
  ]);

  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);

  const viewportRef = useRef<HTMLDivElement>(null);
  const stripRef = useRef<HTMLDivElement>(null);
  const [offset, setOffset] = useState(0);
  const [cardWidth, setCardWidth] = useState(MIN_CARD_WIDTH);

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
      setCardWidth(exactW);
      setOffset(0);
    };

    compute();
    const ro = new ResizeObserver(compute);
    ro.observe(viewport);
    return () => ro.disconnect();
  }, []);

  // Reset visible count and scroll position when filters change
  useEffect(() => {
    setVisibleCount(PAGE_SIZE);
    setOffset(0);
  }, [
    freightTimelineControllerContext.preferredFilter.sources,
    freightTimelineControllerContext.preferredFilter.timerange,
    freightTimelineControllerContext.preferredFilter.warehouses,
    freightTimelineControllerContext.preferredFilter.hideUnusedFreights
  ]);

  const loadMore = useCallback(() => {
    setVisibleCount((prev) => Math.min(prev + PAGE_SIZE, filteredFreights.length));
  }, [filteredFreights.length]);

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
    const hasMore = visibleCount < filteredFreights.length;

    setOffset((prev) => {
      const next = prev + stride;
      if (next > maxOffset) {
        if (hasMore) {
          // Load more so the next page renders; keep the full-stride slide
          loadMore();
          return next;
        }
        return maxOffset;
      }
      return next;
    });
  }, [filteredFreights.length, loadMore, visibleCount]);

  const dndContext = useDndContext();
  const isDragging = !!dndContext.active;

  const [canSlideRight, setCanSlideRight] = useState(false);

  useEffect(() => {
    const viewport = viewportRef.current;
    const strip = stripRef.current;
    if (!viewport || !strip) {
      setCanSlideRight(false);
      return;
    }
    const maxOffset = Math.max(0, strip.scrollWidth - viewport.clientWidth);
    setCanSlideRight(offset < maxOffset - 1 || visibleCount < filteredFreights.length);
  }, [offset, visibleCount, filteredFreights.length]);

  const canSlideLeft = offset > 0;

  return (
    <>
      <div
        className={isPromotionMode ? 'absolute top-0 right-0 bottom-0 left-0 z-20' : ''}
        style={isPromotionMode ? { backgroundColor: 'rgba(0,0,0,.4)' } : {}}
        onClick={() => actionContext?.cancel()}
      />
      {actionContext?.action && (
        <div className='z-20 absolute top-0 left-0 right-0'>
          <BaseHeader>
            <PromotionModeHeader
              loading={getPromotionEligibleFreightQuery.isFetching}
              className='bg-white space-x-2'
            />
          </BaseHeader>
        </div>
      )}
      <div
        className={classNames('freightTimeline', 'bg-white py-2 flex relative z-20')}
        style={{ borderBottom: '2px solid rgba(0,0,0,.05)' }}
      >
        <div
          className='flex items-stretch shrink-0 cursor-pointer select-none text-gray-500 hover:text-gray-700 mx-1'
          style={{ width: ARROW_BUTTON_WIDTH, opacity: canSlideLeft ? 1 : 0.4 }}
          onClick={() => {
            if (canSlideLeft) slideLeft();
          }}
        >
          <div className='m-auto'>
            <FontAwesomeIcon icon={faChevronLeft} />
          </div>
        </div>

        <div
          ref={viewportRef}
          className={classNames('flex-1 min-w-0 relative', {
            'overflow-hidden': !isDragging
          })}
          onWheel={(e) => {
            if (e.deltaY > 0) {
              slideRight();
            } else if (e.deltaY < 0) {
              slideLeft();
            }
          }}
        >
          <div
            ref={stripRef}
            className='flex gap-1 transition-transform duration-300 ease-out'
            style={
              {
                transform: `translateX(-${offset}px)`,
                width: 'max-content',
                '--freight-card-width': `${cardWidth}px`
              } as CSSProperties
            }
          >
            {filteredFreights.slice(0, visibleCount).map((freight) => {
              const freightSoakTime = soakTime?.[freight?.metadata?.name || ''];

              const promotionEligible = Boolean(
                promotionEligibleFreight?.find((f) => f?.metadata?.name === freight?.metadata?.name)
              );

              if (freight.count && freight.count > 0) {
                return (
                  <FreightExpandTile
                    key={`expand-tile-${freight?.metadata?.uid}-${freight?.count}`}
                    count={freight.count}
                  />
                );
              }

              return (
                <FreightCard
                  key={`${freight?.metadata?.uid}-${freight?.count}`}
                  className='h-full'
                  stagesInFreight={
                    dictionaryContext?.freightInStages?.[freight?.metadata?.name || ''] || []
                  }
                  freight={freight}
                  preferredFilter={freightTimelineControllerContext.preferredFilter}
                  setViewingFreight={setViewingFreight}
                  viewingFreight={viewingFreight}
                  inUse={
                    (dictionaryContext?.freightInStages[freight?.metadata?.name || '']?.length ||
                      0) > 0
                  }
                  stageColorMap={colorContext.stageColorMap}
                  promote={isPromotionMode}
                  isPromotionEligibleLoading={getPromotionEligibleFreightQuery.isFetching}
                  promotionEligible={promotionEligible}
                  onReviewAndPromote={() => {
                    const stage = actionContext?.action?.stage?.metadata?.name || '';

                    navigate(
                      generatePath(paths.promote, {
                        name: props.project,
                        freight: freight?.metadata?.name,
                        stage: stage
                      })
                    );
                  }}
                  soakTime={freightSoakTime}
                />
              );
            })}
          </div>
        </div>

        <div
          className='flex items-stretch shrink-0 cursor-pointer select-none text-gray-500 hover:text-gray-700 mx-1'
          style={{ width: ARROW_BUTTON_WIDTH, opacity: canSlideRight ? 1 : 0.4 }}
          onClick={() => {
            if (canSlideRight) slideRight();
          }}
        >
          <div className='m-auto'>
            <FontAwesomeIcon icon={faChevronRight} />
          </div>
        </div>
      </div>
    </>
  );
};
