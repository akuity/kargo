import { useMutation, useQuery } from '@connectrpc/connect-query';
import { message } from 'antd';
import { useEffect, useMemo } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  promoteDownstream,
  promoteToStage,
  queryFreight
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { FreightActionMenu } from '../project/pipelines/freight-action-menu';
import { CollapseMode, FreightTimelineAction } from '../project/pipelines/types';
import { PipelineStateHook, getFreightMode, isPromoting } from '../project/pipelines/utils/state';
import { usePromotionEligibleFreight } from '../project/pipelines/utils/use-promotion-eligible-freight';
import { getSeconds, onError } from '../project/pipelines/utils/util';

import { ConfirmPromotionDialogue } from './confirm-promotion-dialogue';
import { FreightContents } from './freight-contents';
import { FreightItem } from './freight-item';
import { FreightSeparator } from './freight-separator';
import { StageIndicators } from './stage-indicators';

export const FreightTimeline = ({
  freight,
  state,
  stagesPerFreight,
  highlightedStages,
  refetchFreight,
  onHover,
  collapsed,
  setCollapsed,
  stageCount
}: {
  freight: Freight[];
  state: PipelineStateHook;
  promotionEligible: { [key: string]: boolean };
  stagesPerFreight: { [key: string]: Stage[] };
  highlightedStages: { [key: string]: boolean };
  refetchFreight: () => void;
  onHover: (hovering: boolean, freightName: string) => void;
  collapsed: CollapseMode;
  setCollapsed: (collapsed: CollapseMode) => void;
  stageCount: number;
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

  const sortedFreight = useMemo(() => {
    return freight.sort(
      (a, b) =>
        getSeconds(b.metadata?.creationTimestamp) - getSeconds(a.metadata?.creationTimestamp)
    );
  }, [freight]);

  const currentFreight = useMemo(() => {
    let interveningHiddenFreight = 0;
    let seenStages = 0;
    if (!collapsed || collapsed === CollapseMode.Expanded) {
      return sortedFreight.map((f) => ({ freight: f }));
    }

    const filteredFreight = [];

    let i = 0;
    for (const f of sortedFreight) {
      const curStageCount = (stagesPerFreight[f?.metadata?.name || ''] || []).length;
      if (curStageCount === 0) {
        interveningHiddenFreight += 1;
        if (i === sortedFreight.length - 1) {
          filteredFreight.push({ count: interveningHiddenFreight, oldest: true });
        } else if (collapsed === CollapseMode.HideOld && seenStages < stageCount) {
          filteredFreight.push({ freight: f });
        }
      } else {
        seenStages += curStageCount;
        if (collapsed === CollapseMode.HideAll) {
          filteredFreight.push({ count: interveningHiddenFreight });
        }
        interveningHiddenFreight = 0;
        filteredFreight.push({ freight: f });
      }
      i++;
    }

    return filteredFreight;
  }, [freight, collapsed]);

  return (
    <div className='w-full flex h-full'>
      {(currentFreight || []).map((obj, i) => {
        const { freight: f, count, oldest } = obj;
        const id = f?.metadata?.name || `${i}`;
        const curNumStages = (stagesPerFreight[id] || []).length;
        if (count && count > 0) {
          return (
            <FreightSeparator
              key={i}
              count={count}
              onClick={() => setCollapsed(CollapseMode.Expanded)}
              oldest={oldest}
            />
          );
        } else if (f) {
          return (
            <div key={id}>
              <FreightItem
                freight={f || undefined}
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
            </div>
          );
        }
      })}
    </div>
  );
};

export default FreightTimeline;
