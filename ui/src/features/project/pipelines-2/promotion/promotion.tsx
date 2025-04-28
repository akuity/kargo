import { useQuery } from '@connectrpc/connect-query';
import { Descriptions, DescriptionsProps, Drawer, Flex } from 'antd';
import { formatDistance } from 'date-fns';
import { useMemo } from 'react';

import { LoadingState } from '@ui/features/common';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal
} from '@ui/features/common/promotion-status/utils';
import { useDictionaryContext } from '@ui/features/project/pipelines-2/context/dictionary-context';
import { PromotionSteps } from '@ui/features/stage/promotion-steps';
import { hasAbortRequest } from '@ui/features/stage/utils/promotion';
import { getPromotion } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage, Promotion as TPromotion } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { FreightDetails } from './freight-details';
import { getPromotionActor } from './get-promotion-actor';
import { PromotionGraph } from './promotion-graph';
import { useWatchPromotion } from './use-watch-promotion';

type PromotionProps = ModalComponentProps & {
  promotionId: string;
  project: string;
};

const Content = (props: { promotion: TPromotion }) => {
  const dictionaryContext = useDictionaryContext();
  const promotionDescriptions: DescriptionsProps['items'] = [];

  const promotion = props.promotion;

  const isPromotionTerminal = isPromotionPhaseTerminal(getPromotionStatusPhase(promotion));
  const isAbortRequestPending = hasAbortRequest(promotion) && !isPromotionTerminal;

  const freight = useMemo(
    () => dictionaryContext?.freightById?.[promotion?.spec?.freight || ''] as Freight,
    [dictionaryContext?.freightById, promotion]
  );

  const stage = useMemo(
    () => dictionaryContext?.stageByName?.[promotion?.spec?.stage || ''] as Stage,
    [dictionaryContext, promotion]
  );

  if (isAbortRequestPending && promotion?.status) {
    promotion.status.message = 'Promotion Abort Request is in Queue';
  }

  promotionDescriptions.push({
    label: 'status',
    children: (
      <Flex align='center' gap={8}>
        <PromotionStatusIcon
          status={promotion?.status}
          color={isAbortRequestPending ? 'red' : ''}
        />

        <span>{promotion?.status?.phase}</span>
      </Flex>
    )
  });

  const promotionStartTime = timestampDate(promotion?.metadata?.creationTimestamp);

  if (promotionStartTime) {
    const promotionRelativeStartTime = formatDistance(promotionStartTime, new Date(), {
      addSuffix: true
    })?.replace('about', '');

    promotionDescriptions.push({
      label: 'start time',
      children: `${promotionRelativeStartTime}, on ${promotionStartTime}`
    });
  }

  if (isPromotionTerminal) {
    const promotionEndTime = timestampDate(promotion?.status?.finishedAt);

    if (promotionEndTime && promotionStartTime) {
      const duration = formatDistance(promotionStartTime, promotionEndTime, {
        addSuffix: false,
        includeSeconds: true
      })?.replace('about', '');

      promotionDescriptions.push({
        label: 'duration',
        children: <span title={`${promotionEndTime}`}>{duration}</span>
      });
    }
  }

  promotionDescriptions.push({
    label: 'created by',
    children: getPromotionActor(promotion)
  });

  return (
    <>
      <Descriptions
        className='mb-5'
        column={2}
        size='small'
        bordered
        items={promotionDescriptions}
      />

      <div className='my-5'>
        <PromotionSteps promotion={promotion} />
      </div>

      <FreightDetails freight={freight} />

      <PromotionGraph stage={stage} freight={freight} />
    </>
  );
};

export const Promotion = (props: PromotionProps) => {
  const getPromotionQuery = useQuery(getPromotion, {
    project: props.project,
    name: props.promotionId
  });

  useWatchPromotion(props.project, props.promotionId);

  return (
    <Drawer
      open={props.visible}
      onClose={props.hide}
      size='large'
      width={'1224px'}
      title={`Promotion - ${props.promotionId}`}
    >
      {getPromotionQuery.isLoading && <LoadingState />}
      {!getPromotionQuery.isLoading && (
        <Content promotion={getPromotionQuery.data?.result?.value as TPromotion} />
      )}
    </Drawer>
  );
};
