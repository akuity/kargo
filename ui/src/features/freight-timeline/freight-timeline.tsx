import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faArrowsLeftRightToLine } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroller';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  promoteDownstream,
  promoteToStage,
  queryFreight
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { FreightActionMenu } from '../project/pipelines/freight-action-menu';
import { FreightMode, FreightTimelineAction } from '../project/pipelines/types';
import { PipelineStateHook, getFreightMode, isPromoting } from '../project/pipelines/utils/state';
import { usePromotionEligibleFreight } from '../project/pipelines/utils/use-promotion-eligible-freight';
import { getSeconds, onError } from '../project/pipelines/utils/util';

import { ConfirmPromotionDialogue } from './confirm-promotion-dialogue';
import { FreightContents } from './freight-contents';
import { FreightItem } from './freight-item';
import { StageIndicators } from './stage-indicators';

export const FreightTimeline = ({
  freight,
  state,
  stagesPerFreight,
  highlightedStages,
  refetchFreight,
  onHover,
  collapsed,
  setCollapsed
}: {
  freight: Freight[];
  state: PipelineStateHook;
  promotionEligible: { [key: string]: boolean };
  stagesPerFreight: { [key: string]: Stage[] };
  highlightedStages: { [key: string]: boolean };
  refetchFreight: () => void;
  onHover: (hovering: boolean, freightName: string) => void;
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
}) => {
  const navigate = useNavigate();
  const { name: project } = useParams();

  const { mutate: promoteDownstreamAction } = useMutation(promoteDownstream, {
    onError,
    onSuccess: () => {
      message.success(
        `Promotion requests to all subscribers of "${state.stage}" have been submitted.`
      );
      state.clear();
    }
  });

  const { mutate: promoteAction } = useMutation(promoteToStage, {
    onError,
    onSuccess: () => {
      message.success(
        `Promotion request for stage "${state.stage}" has been successfully submitted.`
      );
      state.clear();
    }
  });

  const {
    data: availableFreightData,
    refetch: refetchAvailableFreight,
    isLoading: isLoadingAvailableFreight
  } = useQuery(queryFreight, { project, stage: state.stage || '' });

  const promotionEligible = usePromotionEligibleFreight(
    availableFreightData?.groups['']?.freight || [],
    state.action,
    state.stage,
    isLoadingAvailableFreight || !isPromoting(state)
  );

  useEffect(() => {
    if (!isPromoting(state)) {
      return;
    }
    refetchAvailableFreight();
  }, [state.action, state.stage, freight]);

  const [loadedItems, setLoadedItems] = useState(20);
  const loadFunc = (loadedLength: number) => {
    setLoadedItems((length) => length + loadedLength);
  };

  const currentFreight = freight.slice(0, loadedItems);

  let seenStages = 0;
  let displayedCollapsed = false;
  const numStages = useMemo(() => {
    return Object.keys(stagesPerFreight).reduce(
      (acc, cur) => (cur?.length > 0 ? acc + stagesPerFreight[cur].length : acc),
      0
    );
  }, [stagesPerFreight]);

  return (
    <>
      <InfiniteScroll
        pageStart={0}
        loadMore={loadFunc}
        className='w-full flex h-full'
        hasMore={freight.length > currentFreight.length}
      >
        {(currentFreight || [])
          .sort(
            (a, b) =>
              getSeconds(b.metadata?.creationTimestamp) - getSeconds(a.metadata?.creationTimestamp)
          )
          .map((f, i) => {
            const id = f?.metadata?.name || `${i}`;
            const curNumStages = (stagesPerFreight[id] || []).length;
            if (curNumStages > 0) {
              seenStages += curNumStages;
            }
            if (seenStages >= numStages && curNumStages === 0 && collapsed) {
              const tmp = displayedCollapsed;
              displayedCollapsed = true;
              return tmp ? null : (
                <FreightItem
                  onClick={() => setCollapsed(false)}
                  empty={true}
                  highlighted={false}
                  key='collapsed'
                  mode={FreightMode.Default}
                  onHover={() => null}
                  hideLabel={true}
                >
                  <FontAwesomeIcon
                    icon={faArrowsLeftRightToLine}
                    className='text-gray-300'
                    size='2x'
                  />
                </FreightItem>
              );
            }
            return (
              <FreightItem
                freight={f || undefined}
                key={i}
                onClick={() => {
                  if (state.stage && promotionEligible[id]) {
                    state.select(undefined, undefined, id);
                  } else {
                    navigate(generatePath(paths.freight, { name: project, freightName: id }));
                  }
                }}
                mode={getFreightMode(state, id, promotionEligible[id])}
                empty={curNumStages === 0}
                onHover={(h) => onHover(h, id)}
                highlighted={
                  state.action === FreightTimelineAction.PromoteFreight
                    ? state.freight === f?.metadata?.name
                    : (stagesPerFreight[id] || []).reduce((h, cur) => {
                        if (h) {
                          return true;
                        }
                        return highlightedStages[cur.metadata?.name || ''];
                      }, false)
                }
              >
                {!state.action && (
                  <FreightActionMenu
                    freight={f}
                    approveAction={() => {
                      state.select(FreightTimelineAction.ManualApproval, undefined, id);
                    }}
                    refetchFreight={refetchFreight}
                    inUse={stagesPerFreight[id]?.length > 0}
                    promoteAction={() => {
                      state.select(FreightTimelineAction.PromoteFreight, undefined, id);
                    }}
                  />
                )}
                <StageIndicators
                  stages={stagesPerFreight[id] || []}
                  faded={state.action === FreightTimelineAction.ManualApproval}
                />
                <FreightContents
                  highlighted={
                    // contains stages, not in promotion mode
                    ((stagesPerFreight[id] || []).length > 0 && !isPromoting(state)) ||
                    // in promotion mode, is eligible
                    (isPromoting(state) && promotionEligible[id]) ||
                    false
                  }
                  freight={f}
                />
                {isPromoting(state) && state.freight === id && (
                  <ConfirmPromotionDialogue
                    stageName={state.stage || ''}
                    promotionType={state.action || 'default'}
                    onClick={() => {
                      const currentData = {
                        project,
                        freight: f?.metadata?.name
                      };
                      if (state.action === FreightTimelineAction.Promote) {
                        promoteAction({
                          stage: state.stage || '',
                          ...currentData
                        });
                      } else {
                        promoteDownstreamAction({
                          stage: state.stage || '',
                          ...currentData
                        });
                      }
                    }}
                  />
                )}
              </FreightItem>
            );
          })}
      </InfiniteScroll>
    </>
  );
};

export default FreightTimeline;
