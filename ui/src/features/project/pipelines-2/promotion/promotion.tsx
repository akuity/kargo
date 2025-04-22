import { Descriptions, DescriptionsProps, Drawer, Flex } from 'antd';
import { formatDistance } from 'date-fns';
import { useMemo } from 'react';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal
} from '@ui/features/common/promotion-status/utils';
import { useDictionaryContext } from '@ui/features/project/pipelines-2/context/dictionary-context';
import { PromotionSteps } from '@ui/features/stage/promotion-steps';
import { hasAbortRequest } from '@ui/features/stage/utils/promotion';
import { Freight, Stage, Promotion as TPromotion } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { FreightDetails } from './freight-details';
import { getPromotionActor } from './get-promotion-actor';
import { PromotionGraph } from './promotion-graph';

type PromotionProps = ModalComponentProps & {
  promotion: TPromotion;
};

export const Promotion = (props: PromotionProps) => {
  const dictionaryContext = useDictionaryContext();
  const promotionDescriptions: DescriptionsProps['items'] = [];

  const isPromotionTerminal = isPromotionPhaseTerminal(getPromotionStatusPhase(props.promotion));
  const isAbortRequestPending = hasAbortRequest(props.promotion) && !isPromotionTerminal;

  const freight = useMemo(
    () => dictionaryContext?.freightById?.[props.promotion?.spec?.freight || ''] as Freight,
    [dictionaryContext?.freightById, props.promotion]
  );

  const stage = useMemo(
    () => dictionaryContext?.stageByName?.[props.promotion?.spec?.stage || ''] as Stage,
    [dictionaryContext, props.promotion]
  );

  if (isAbortRequestPending && props.promotion?.status) {
    props.promotion.status.message = 'Promotion Abort Request is in Queue';
  }

  promotionDescriptions.push({
    label: 'status',
    children: (
      <Flex align='center' gap={8}>
        <PromotionStatusIcon
          status={props.promotion?.status}
          color={isAbortRequestPending ? 'red' : ''}
        />

        <span>{props.promotion?.status?.phase}</span>
      </Flex>
    )
  });

  const promotionStartTime = timestampDate(props.promotion?.metadata?.creationTimestamp);

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
    const promotionEndTime = timestampDate(props.promotion?.status?.finishedAt);

    if (promotionEndTime && promotionStartTime) {
      const duration = formatDistance(promotionStartTime, promotionEndTime, {
        addSuffix: false
      })?.replace('about', '');

      promotionDescriptions.push({
        label: 'duration',
        children: <span title={`${promotionEndTime}`}>{duration}</span>
      });
    }
  }

  promotionDescriptions.push({
    label: 'created by',
    children: getPromotionActor(props.promotion)
  });

  return (
    <Drawer
      open={props.visible}
      onClose={props.hide}
      size='large'
      width={'1224px'}
      title={`Promotion - ${props.promotion?.metadata?.name}`}
    >
      <Descriptions
        className='mb-5'
        column={2}
        size='small'
        bordered
        items={promotionDescriptions}
      />

      <div className='my-5'>
        <PromotionSteps promotion={props.promotion} />
      </div>

      <FreightDetails freight={freight} />

      <PromotionGraph stage={stage} freight={freight} />
    </Drawer>
  );
};
