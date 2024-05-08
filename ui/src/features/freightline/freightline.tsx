import { useMutation, useQuery } from '@connectrpc/connect-query';
import { message } from 'antd';
import React, { useEffect } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  promoteToStage,
  promoteToStageSubscribers,
  queryFreight
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { FreightActionMenu } from '../project/pipelines/freight-action-menu';
import { FreightlineAction } from '../project/pipelines/types';
import { PipelineStateHook, getFreightMode, isPromoting } from '../project/pipelines/utils/state';
import { usePromotionEligibleFreight } from '../project/pipelines/utils/use-promotion-eligible-freight';
import { getSeconds, onError } from '../project/pipelines/utils/util';

import { ConfirmPromotionDialogue } from './confirm-promotion-dialogue';
import { FreightContents } from './freight-contents';
import { FreightItem } from './freight-item';
import { FreightlineHeader } from './freightline-header';
import { StageIndicators } from './stage-indicators';

export const Freightline = ({
  freight,
  state,
  subscribersByStage,
  stagesPerFreight,
  highlightedStages,
  refetchFreight,
  onHover
}: {
  freight: Freight[];
  state: PipelineStateHook;
  promotionEligible: { [key: string]: boolean };
  subscribersByStage: { [key: string]: Stage[] };
  stagesPerFreight: { [key: string]: Stage[] };
  highlightedStages: { [key: string]: boolean };
  refetchFreight: () => void;
  onHover: (hovering: boolean, freightName: string) => void;
}) => {
  const navigate = useNavigate();
  const { name: project } = useParams();

  const { mutate: promoteToStageSubscribersAction } = useMutation(promoteToStageSubscribers, {
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
    state.stage,
    isLoadingAvailableFreight || !isPromoting(state)
  );

  useEffect(() => {
    if (!isPromoting(state)) {
      return;
    }
    refetchAvailableFreight();
  }, [state.action, state.stage, freight]);

  return (
    <>
      <FreightlineHeader
        promotingStage={state.stage}
        action={state.action}
        cancel={state.clear}
        downstreamSubs={(subscribersByStage[state.stage || ''] || []).map(
          (s) => s.metadata?.name || ''
        )}
      />
      <FreightlineWrapper>
        <>
          {(freight || [])
            .sort(
              (a, b) =>
                getSeconds(b.metadata?.creationTimestamp) -
                getSeconds(a.metadata?.creationTimestamp)
            )
            .map((f, i) => {
              const id = f?.metadata?.name || `${i}`;
              return (
                <FreightItem
                  freight={f || undefined}
                  key={id}
                  onClick={() => {
                    if (state.stage && promotionEligible[id]) {
                      state.select(undefined, undefined, id);
                    } else {
                      navigate(generatePath(paths.freight, { name: project, freightName: id }));
                    }
                  }}
                  mode={getFreightMode(state, id, promotionEligible[id])}
                  empty={(stagesPerFreight[id] || []).length === 0}
                  onHover={(h) => onHover(h, id)}
                  highlighted={(stagesPerFreight[id] || []).reduce((h, cur) => {
                    if (h) {
                      return true;
                    }
                    return highlightedStages[cur.metadata?.name || ''];
                  }, false)}
                >
                  <FreightActionMenu
                    freight={f}
                    approveAction={() => {
                      state.select(FreightlineAction.ManualApproval, undefined, id);
                    }}
                    refetchFreight={refetchFreight}
                  />
                  <StageIndicators
                    stages={stagesPerFreight[id] || []}
                    faded={state.action === FreightlineAction.ManualApproval}
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
                        if (state.action === FreightlineAction.Promote) {
                          promoteAction({
                            stage: state.stage || '',
                            ...currentData
                          });
                        } else {
                          promoteToStageSubscribersAction({
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
        </>
      </FreightlineWrapper>
    </>
  );
};

const FreightlineWrapper = ({ children }: { children: React.ReactNode }) => {
  return (
    <div className='w-full py-3 flex flex-col overflow-hidden' style={{ backgroundColor: '#eee' }}>
      <div className='flex h-44 w-full items-center px-1'>
        <div
          className='text-gray-500 text-sm font-semibold mb-2 w-min h-min'
          style={{ transform: 'rotate(-0.25turn)' }}
        >
          NEW
        </div>
        <div className='flex items-center h-full overflow-x-auto'>{children}</div>
        <div className='rotate-90 text-gray-500 text-sm font-semibold ml-auto'>OLD</div>
      </div>
    </div>
  );
};
