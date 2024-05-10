import { useMutation, useQuery } from '@connectrpc/connect-query';
import { message } from 'antd';
import { useEffect, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroller';
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
import { StageIndicators } from './stage-indicators';

// const x = Freight.fromJsonString(
//   `{"metadata":{"name":"c760855def6ac71133eefa2ec21bd3e53561e323","generateName":"","namespace":"kargo-demo","selfLink":"","uid":"ee293ccb-28c9-464d-9f81-37af8ab638c2","resourceVersion":"5471978","generation":"1","creationTimestamp":"2024-05-08T23:42:33Z","labels":{"kargo.akuity.io/alias":"erstwhile-gorilla"},"annotations":{},"ownerReferences":[],"finalizers":[],"managedFields":[{"manager":"kargo","operation":"Update","apiVersion":"kargo.akuity.io/v1alpha1","time":"2024-05-08T23:42:33Z","fieldsType":"FieldsV1","fieldsV1":{"Raw":"eyJmOmNvbW1pdHMiOnt9LCJmOmltYWdlcyI6e30sImY6d2FyZWhvdXNlIjp7fX0="},"subresource":""}]},"commits":[{"repoURL":"https://github.com/jessesuen/kargo-advanced","id":"32be5f08c2f16adb2fbcb874b6e2d9d3537c8717","branch":"","tag":"","healthCheckCommit":"","message":"TEST=test","author":""}],"images":[{"repoURL":"nginx","gitRepoURL":"","tag":"1.26.0","digest":"sha256:9f0d283eccddedf25816104877faf1cb584a8236ec4d7985a4965501d080d84f"}],"charts":[],"status":{"verifiedIn":{},"approvedFor":{}},"alias":"erstwhile-gorilla","warehouse":"kargo-demo"}`
// );

// const y = Array(100).fill(x as Freight) as Freight[];
// const freight = y;

export const Freightline = ({
  freight,
  state,
  stagesPerFreight,
  highlightedStages,
  refetchFreight,
  onHover
}: {
  freight: Freight[];
  state: PipelineStateHook;
  promotionEligible: { [key: string]: boolean };
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

  const [loadedItems, setLoadedItems] = useState(20);
  const loadFunc = (loadedLength: number) => {
    setLoadedItems((length) => length + loadedLength);
  };

  const currentFreight = freight.slice(0, loadedItems);

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
      </InfiniteScroll>
    </>
  );
};

export default Freightline;
