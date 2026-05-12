import { useDndContext } from '@dnd-kit/core';
import { faChevronLeft, faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';
import { CSSProperties, useContext, useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { BaseHeader } from '@ui/features/common/layout/base-header';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { useFreightTimelineControllerContext } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { FreightCard } from './freight-card';
import { FreightExpandTile } from './freight-expand-tile';
import { PromotionModeHeader } from './promotion-mode-header';
import { useFilteredFreights } from './use-filtered-freights';
import { useFreightCarousel } from './use-freight-carousel';
import { usePromotionEligibleFreight } from './use-promotion-eligible-freight';
import { useSoakTime } from './use-soak-time';

import './freight-timeline.less';

const ARROW_BUTTON_WIDTH = 28;

export const FreightTimeline = (props: { freights: Freight[]; project: string }) => {
  const navigate = useNavigate();

  const colorContext = useContext(ColorContext);
  const freightTimelineControllerContext = useFreightTimelineControllerContext();
  const dictionaryContext = useDictionaryContext();
  const actionContext = useActionContext();

  if (!freightTimelineControllerContext) {
    throw new Error('missing context freightTimelineControllerContext');
  }

  const { preferredFilter } = freightTimelineControllerContext;

  const isPromotionMode =
    actionContext?.action?.type === IAction.PROMOTE ||
    actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM;

  const { promotionEligibleFreight, ...getPromotionEligibleFreightQuery } =
    usePromotionEligibleFreight(props.project);

  const soakTime = useSoakTime(props.freights);

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
  } = useFreightCarousel(filteredFreights.length, preferredFilter);

  const dndContext = useDndContext();
  const isDragging = !!dndContext.active;

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
                  preferredFilter={preferredFilter}
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
