import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faRefresh, faStopCircle, faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Descriptions, DescriptionsProps, Drawer, Flex, message, Modal, Tabs } from 'antd';
import { formatDistance } from 'date-fns';
import { useMemo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  isPromotionRetryable
} from '@ui/features/common/promotion-status/utils';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { PromotionSteps } from '@ui/features/stage/promotion-steps';
import { canAbortPromotion, hasAbortRequest } from '@ui/features/stage/utils/promotion';
import {
  abortPromotion,
  getPromotion,
  promoteToStage,
  refreshStage
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight, Promotion as TPromotion } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import { FreightDetails } from './freight-details';
import { getPromotionActor } from './get-promotion-actor';
import { getPromotionStage } from './get-promotion-stage';
import { useWatchPromotion } from './use-watch-promotion';

type PromotionProps = ModalComponentProps & {
  promotionId: string;
  project: string;
};

const Content = (props: { promotion: TPromotion; yaml: string }) => {
  const navigate = useNavigate();
  const dictionaryContext = useDictionaryContext();
  const promotionDescriptions: DescriptionsProps['items'] = [];

  const abortPromotionMutation = useMutation(abortPromotion, {
    onSuccess: () =>
      // Abort promotion annotates the Promotion resource and then controller acts
      message.success({
        content: `Abort Promotion ${promotion.metadata?.name} requested successfully.`
      })
  });

  const promoteMutation = useMutation(promoteToStage, {
    onSuccess(data) {
      navigate(
        generatePath(paths.promotion, {
          name: props.promotion?.metadata?.namespace,
          promotionId: data.promotion?.metadata?.name
        })
      );
    }
  });

  const refreshStageMutation = useMutation(refreshStage);

  const promotion = props.promotion;
  const affiliatedStage = getPromotionStage(promotion);

  const onRetryPromotion = () => {
    const stage = affiliatedStage;
    const project = props.promotion?.metadata?.namespace;
    const freight = promotion?.spec?.freight;

    promoteMutation.mutate({
      stage,
      project,
      freight
    });
  };

  const promotionStatusPhase = getPromotionStatusPhase(promotion);
  const isPromotionTerminal = isPromotionPhaseTerminal(promotionStatusPhase);
  const canRetry = isPromotionRetryable(promotionStatusPhase);
  const isAbortRequestPending = hasAbortRequest(promotion) && !isPromotionTerminal;

  const freight = useMemo(
    () => dictionaryContext?.freightById?.[promotion?.spec?.freight || ''] as Freight,
    [dictionaryContext?.freightById, promotion]
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

        {canRetry && (
          <Button
            loading={promoteMutation.isPending}
            size='small'
            icon={<FontAwesomeIcon icon={faUndo} />}
            onClick={onRetryPromotion}
          >
            Retry
          </Button>
        )}
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

  const confirmAbortRequest = () =>
    Modal.confirm({
      width: '656px',
      icon: <FontAwesomeIcon icon={faStopCircle} className='text-lg text-red-500 mr-5' />,
      title: 'Abort Promotion Request',
      onOk: () =>
        abortPromotionMutation.mutate({
          project: promotion?.metadata?.namespace,
          name: promotion?.metadata?.name
        }),
      okText: 'Abort',
      okButtonProps: {
        danger: true
      },
      content: (
        <Descriptions
          size='small'
          className='mt-2'
          column={1}
          bordered
          items={[
            {
              key: 'name',
              label: 'Name',
              children: promotion.metadata?.name
            },
            {
              key: 'date',
              label: 'Start Date',
              children: timestampDate(promotion.metadata?.creationTimestamp)?.toString()
            }
          ]}
        />
      )
    });

  return (
    <>
      {affiliatedStage && (
        <Flex align='center' gap={8} className='mb-2'>
          <h1 className='text-sm font-semibold m-0'>Stage: </h1>
          {affiliatedStage}
          <Button
            size='small'
            icon={<FontAwesomeIcon icon={faRefresh} size='1x' />}
            onClick={() =>
              refreshStageMutation.mutate({
                project: promotion?.metadata?.namespace,
                name: affiliatedStage
              })
            }
            loading={refreshStageMutation.isPending}
          >
            Refresh
          </Button>
        </Flex>
      )}
      <Tabs
        className='mb-5'
        items={[
          {
            key: 'promotion',
            label: 'Promotion',
            children: (
              <>
                <Descriptions
                  className='mb-5'
                  column={2}
                  size='small'
                  bordered
                  items={promotionDescriptions}
                />

                <div className='mt-5'>
                  {canAbortPromotion(promotion) && (
                    <Flex className='mb-2'>
                      <Button
                        className='ml-auto'
                        danger
                        size='small'
                        onClick={confirmAbortRequest}
                        icon={<FontAwesomeIcon icon={faStopCircle} className='text-xs' />}
                      >
                        Abort
                      </Button>
                    </Flex>
                  )}
                  <PromotionSteps promotion={promotion} />
                </div>
              </>
            )
          },
          {
            key: 'yaml',
            label: 'YAML',
            children: (
              <YamlEditor value={props.yaml} height='500px' disabled isHideManagedFieldsDisplayed />
            )
          }
        ]}
      />

      <FreightDetails freight={freight} />
    </>
  );
};

export const Promotion = (props: PromotionProps) => {
  const getPromotionQuery = useQuery(getPromotion, {
    project: props.project,
    name: props.promotionId
  });

  const rawPromotionYamlQuery = useQuery(getPromotion, {
    project: props.project,
    name: props.promotionId,
    format: RawFormat.YAML
  });

  useWatchPromotion(props.project, props.promotionId);

  const rawPromotionYaml = useMemo(
    () => decodeRawData(rawPromotionYamlQuery.data),
    [rawPromotionYamlQuery.data]
  );

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
        <Content
          promotion={getPromotionQuery.data?.result?.value as TPromotion}
          yaml={rawPromotionYaml}
        />
      )}
    </Drawer>
  );
};
