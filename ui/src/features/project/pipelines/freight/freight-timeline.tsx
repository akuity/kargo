import { faCaretLeft, faCaretRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Divider } from 'antd';
import classNames from 'classnames';
import { useContext, useMemo, useRef, useState } from 'react';
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
import { FreightTimelineFilters } from './freight-timeline-filters';
import { PromotionModeHeader } from './promotion-mode-header';
import { filterFreightBySource, filterFreightByTimerange } from './source-catalogue-utils';
import { usePromotionEligibleFreight } from './use-promotion-eligible-freight';
import './freight-timeline.less';

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

  if (!freightTimelineControllerContext) {
    throw new Error('missing context freightTimelineControllerContext');
  }

  const [filtersCollapsed, setFilterCollapsed] = useState(true);

  const [viewingFreight, setViewingFreight] = useState<Freight | null>(null);

  const filteredFreights: (Freight & {
    count?: number;
  })[] = useMemo(() => {
    let filtered = props.freights?.sort((a, b) => {
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

    return filtered;
  }, [
    props.freights,
    freightTimelineControllerContext.preferredFilter.sources,
    freightTimelineControllerContext.preferredFilter.timerange,
    freightTimelineControllerContext.preferredFilter.warehouses,
    freightTimelineControllerContext.preferredFilter.hideUnusedFreights,
    dictionaryContext?.freightInStages
  ]);

  const freightListStyleRef = useRef<HTMLDivElement>(null);

  const scrollCarouselLeft = () => {
    const right = freightListStyleRef.current?.style.right || '0px';

    let nextRight = +right.slice(0, -2) - 80;

    if (nextRight < 0) {
      nextRight = 0;
    }

    freightListStyleRef.current?.style.setProperty('right', `${nextRight}px`);
  };

  const scrollCarouselRight = () => {
    const right = freightListStyleRef.current?.style.right || '0px';

    const nextRight = +right.slice(0, -2) + 80;

    if (nextRight >= (freightListStyleRef.current?.clientWidth || 0) / 2) {
      return;
    }

    freightListStyleRef.current?.style.setProperty('right', `${nextRight}px`);
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
              className='bg-white space-x-2'
            />
          </BaseHeader>
        </div>
      )}
      <div
        className={classNames('freightTimeline', 'bg-white pl-4 py-2 flex gap-3 z-20')}
        style={{ borderBottom: '2px solid rgba(0,0,0,.05)' }}
      >
        <FreightTimelineFilters
          collapsed={filtersCollapsed}
          filteredFreights={filteredFreights}
          freights={props.freights}
          onCollapseToggle={() => setFilterCollapsed(!filtersCollapsed)}
          onPreferredFilterChange={freightTimelineControllerContext.setPreferredFilter}
          preferredFilter={freightTimelineControllerContext.preferredFilter}
        />
        {!filtersCollapsed && <Divider type='vertical' className='h-full' />}
        <div
          className='w-full flex overflow-hidden relative px-5'
          onWheel={(e) => {
            if (e.deltaX > 0) {
              scrollCarouselRight();
              return;
            }

            scrollCarouselLeft();
          }}
        >
          <div className='flex gap-1 relative transition-all right-0' ref={freightListStyleRef}>
            {filteredFreights.map((freight) => {
              const promotionEligible = Boolean(
                promotionEligibleFreight?.find((f) => f?.metadata?.name === freight?.metadata?.name)
              );

              return (
                <FreightCard
                  key={`${freight?.metadata?.uid}-${freight?.count}`}
                  count={freight?.count}
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
                  promote={isPromotionMode && promotionEligible}
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
                  onExpand={() =>
                    freightTimelineControllerContext?.setPreferredFilter({
                      ...freightTimelineControllerContext?.preferredFilter,
                      hideUnusedFreights: false
                    })
                  }
                />
              );
            })}
          </div>

          <div
            className='absolute left-0 h-full bg-gray-100 px-1 flex items-center cursor-pointer'
            onClick={() => {
              scrollCarouselLeft();
            }}
          >
            <FontAwesomeIcon icon={faCaretLeft} />
          </div>

          <div
            className='absolute right-0 h-full bg-gray-100 px-1 flex items-center cursor-pointer'
            onClick={() => {
              scrollCarouselRight();
            }}
          >
            <FontAwesomeIcon icon={faCaretRight} />
          </div>
        </div>
      </div>
    </>
  );
};
