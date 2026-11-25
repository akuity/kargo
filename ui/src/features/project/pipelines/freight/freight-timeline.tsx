import { useDndContext } from '@dnd-kit/core';
import { faArrowLeft, faChevronLeft, faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';
import { CSSProperties, useContext, useMemo, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { WarehouseExpanded } from '@ui/extend/types';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { useFreightTimelineControllerContext } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v2/models';

import { getPreviousFreight } from './freight-artifact-changes-utils';
import { FreightCard } from './freight-card';
import { FreightExpandTile } from './freight-expand-tile';
import { FreightTimelineEmpty } from './freight-timeline-empty';
import { FreightWarehouseGroup } from './freight-warehouse-group';
import {
  getFocusedWarehouse,
  groupByWarehouse,
  takeFreight
} from './freight-warehouse-group-utils';
import { PromotionModeHeader } from './promotion-mode-header';
import { getShowSubscriptionNames } from './subscription-label-utils';
import { FilteredFreight, useFilteredFreights } from './use-filtered-freights';
import { useFreightCarousel } from './use-freight-carousel';
import { usePromotionEligibleFreight } from './use-promotion-eligible-freight';
import { useSoakTime } from './use-soak-time';

import './freight-timeline.less';

const ARROW_BUTTON_WIDTH = 28;

// Sits left of the viewport, as far from the Warehouse header's icon as the
// header's own gap (6px), over the left scroll arrow's column.
const backButtonClassName =
  'absolute right-full top-0 mr-1.5 z-10 flex items-center justify-center w-5 h-5 p-0 border-0 bg-transparent cursor-pointer text-xs text-[color:var(--kargo-color-text-secondary)] hover:text-[color:var(--kargo-color-text)]';

export const FreightTimeline = (props: {
  freights: Freight[];
  project: string;
  warehouses: WarehouseExpanded[];
}) => {
  const navigate = useNavigate();

  const colorContext = useContext(ColorContext);
  const freightTimelineControllerContext = useFreightTimelineControllerContext();
  const dictionaryContext = useDictionaryContext();
  const actionContext = useActionContext();

  if (!freightTimelineControllerContext) {
    throw new Error('missing context freightTimelineControllerContext');
  }

  const { preferredFilter, setPreferredFilter } = freightTimelineControllerContext;

  const isPromotionMode =
    actionContext?.action?.type === IAction.PROMOTE ||
    actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM;

  // the Warehouse the freight timeline is narrowed to, if any
  const focusedWarehouse = getFocusedWarehouse(
    preferredFilter.freightTimelineWarehouse,
    isPromotionMode
  );
  const isFocused = Boolean(focusedWarehouse);

  const setFocusedWarehouse = (freightTimelineWarehouse: string) =>
    setPreferredFilter({ ...preferredFilter, freightTimelineWarehouse });

  const { promotionEligibleFreight, ...getPromotionEligibleFreightQuery } =
    usePromotionEligibleFreight(props.project);

  const soakTime = useSoakTime(props.freights);

  // From every piece of freight rather than the filtered ones, so filtering
  // the timeline does not change what a card's artifacts are compared with.
  const previousFreight = useMemo(() => getPreviousFreight(props.freights), [props.freights]);

  // A Map rather than an object: freight from a deleted Warehouse named, say,
  // "constructor" would otherwise read what objects inherit.
  const subscriptionsByWarehouse = useMemo(
    () =>
      new Map(
        props.warehouses.map((warehouse) => [
          warehouse?.metadata?.name || '',
          warehouse?.spec?.subscriptions || []
        ])
      ),
    [props.warehouses]
  );

  const showSubscriptionNames = getShowSubscriptionNames(
    preferredFilter.showSubscriptionNames,
    props.warehouses
  );

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const filteredFreights = useFilteredFreights(props.freights, preferredFilter);

  const {
    viewportRef,
    stripRef,
    cardWidth,
    offset,
    visibleCount,
    slideLeft,
    slideRight,
    canSlideLeft,
    canSlideRight
  } = useFreightCarousel(filteredFreights.length, preferredFilter, isPromotionMode);

  const dndContext = useDndContext();
  const isDragging = !!dndContext.active;

  const isEmpty = filteredFreights.length === 0;
  const groups = groupByWarehouse(filteredFreights);
  // Name each Warehouse above its freight only when there is more than one.
  // A project with a single Warehouse can still hold freight left by a
  // deleted one, which then needs its name too.
  const withHeader = props.warehouses.length > 1 || groups.length > 1;

  const filtersActive =
    preferredFilter.sources.length > 0 ||
    preferredFilter.warehouses.length > 0 ||
    preferredFilter.timerange !== 'all-time' ||
    preferredFilter.hideUnusedFreights ||
    isFocused;

  const renderFreight = (freight: FilteredFreight) => {
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
        stagesInFreight={dictionaryContext?.freightInStages?.[freight?.metadata?.name || ''] || []}
        freight={freight}
        preferredFilter={preferredFilter}
        setViewingFreight={setViewingFreight}
        viewingFreight={viewingFreight}
        inUse={(dictionaryContext?.freightInStages[freight?.metadata?.name || '']?.length || 0) > 0}
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
        previousFreight={previousFreight[freight?.metadata?.name || '']}
        showSubscriptionNames={showSubscriptionNames}
        subscriptions={subscriptionsByWarehouse.get(freight?.origin?.name || '')}
      />
    );
  };

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
              className='bg-[var(--kargo-topbar-bg)] space-x-2'
            />
          </BaseHeader>
        </div>
      )}
      <div
        className={classNames(
          'freightTimeline',
          'bg-[var(--kargo-topbar-bg)] py-2 flex relative z-20'
        )}
        style={{ borderBottom: '2px solid var(--kargo-color-border-secondary, rgba(0,0,0,.05))' }}
      >
        {isEmpty && isFocused && (
          // room for the back arrow, where the left scroll arrow would be
          <div className='shrink-0 mx-1' style={{ width: ARROW_BUTTON_WIDTH }} />
        )}

        {!isEmpty && (
          <div
            className='flex items-stretch shrink-0 cursor-pointer select-none text-gray-500 dark:text-neutral-400 hover:text-gray-700 dark:hover:text-neutral-200 mx-1'
            style={{ width: ARROW_BUTTON_WIDTH, opacity: canSlideLeft ? 1 : 0.4 }}
            onClick={() => {
              if (canSlideLeft) slideLeft();
            }}
          >
            <div className='m-auto'>
              <FontAwesomeIcon icon={faChevronLeft} />
            </div>
          </div>
        )}

        <div className='flex-1 min-w-0 relative'>
          {isFocused && (
            // Just outside the viewport, which would clip it, level with the
            // Warehouse header, so the header's icon stays in line with the
            // cards as it does across all Warehouses. Shown when no freight
            // is left too, as the way back.
            <button
              type='button'
              aria-label='Show freight from all Warehouses'
              title='Show freight from all Warehouses'
              className={backButtonClassName}
              onClick={() => setFocusedWarehouse('')}
            >
              <FontAwesomeIcon icon={faArrowLeft} />
            </button>
          )}

          <div
            ref={viewportRef}
            className={classNames('h-full relative', {
              'overflow-hidden': !isDragging
            })}
          >
            {isEmpty && (
              <FreightTimelineEmpty
                filtered={filtersActive}
                hasWarehouses={props.warehouses.length > 0}
                promotionMode={isPromotionMode}
              />
            )}

            <div
              ref={stripRef}
              // positioned, so each group can measure where it sits in the strip
              className='relative flex gap-1 transition-transform duration-300 ease-out'
              style={
                {
                  transform: `translateX(-${offset}px)`,
                  width: 'max-content',
                  '--freight-card-width': `${cardWidth}px`
                } as CSSProperties
              }
            >
              {takeFreight(groups, visibleCount).map((group) => (
                <FreightWarehouseGroup
                  key={group.key}
                  warehouse={group.warehouse}
                  withHeader={withHeader}
                  color={colorContext.warehouseColorMap?.[group.warehouse]}
                  offset={offset}
                  cardWidth={cardWidth}
                  onShowOnly={
                    // while promoting, the timeline shows what the Stage can
                    // take, whichever Warehouse it was narrowed to
                    isFocused || isPromotionMode || !group.warehouse
                      ? undefined
                      : () => setFocusedWarehouse(group.warehouse)
                  }
                  focused={isFocused}
                >
                  {group.freights.map(renderFreight)}
                </FreightWarehouseGroup>
              ))}
            </div>
          </div>
        </div>

        {!isEmpty && (
          <div
            className='flex items-stretch shrink-0 cursor-pointer select-none text-gray-500 dark:text-neutral-400 hover:text-gray-700 dark:hover:text-neutral-200 mx-1'
            style={{ width: ARROW_BUTTON_WIDTH, opacity: canSlideRight ? 1 : 0.4 }}
            onClick={() => {
              if (canSlideRight) slideRight();
            }}
          >
            <div className='m-auto'>
              <FontAwesomeIcon icon={faChevronRight} />
            </div>
          </div>
        )}
      </div>
    </>
  );
};
