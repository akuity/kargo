import { useMutation } from '@connectrpc/connect-query';
import { useDroppable } from '@dnd-kit/core';
import {
  faBarsStaggered,
  faBolt,
  faExternalLink,
  faMinus,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Dropdown, Flex, message, Space, Typography } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { ReactNode, useMemo } from 'react';
import { generatePath, Link, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { ArgoCDLink } from '@ui/features/project/pipelines/nodes/argocd-link';
import {
  approveFreight,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { useDictionaryContext } from '../context/dictionary-context';
import { useGraphContext } from '../context/graph-context';
import { stageIndexer } from '../graph/node-indexer';
import { DropOverlay } from '../promotion/drag-and-drop/drop-overlay';

import { AnalysisRunLogsLink } from './analysis-run-logs-link';
import style from './node-size-source-of-truth.module.less';
import { useGetPromotionDropdownItems } from './promotion/use-get-promotion-dropdown-items';
import { PullRequestLink } from './pull-request-link';
import { StageFreight } from './stage-freight';
import {
  getLastPromotionDate,
  getStageHealth,
  isStageControlFlow,
  useHideStageIfInPromotionMode,
  useStageHeaderStyle
} from './stage-meta-utils';
import { StageNodePhase } from './stage-node-phase';

import './stage-node.less';

export const StageNode = (props: { stage: Stage }) => {
  const stageName = props.stage?.metadata?.name || '';

  const navigate = useNavigate();
  const dictionaryContext = useDictionaryContext();
  const graphContext = useGraphContext();
  const actionContext = useActionContext();

  const stageNodeIndex = useMemo(() => stageIndexer.index(props.stage), [props.stage]);

  const headerStyle = useStageHeaderStyle(props.stage);

  const autoPromotionMode = dictionaryContext?.stageAutoPromotionMap?.[stageName];

  const stageHealth = getStageHealth(props.stage);

  const controlFlow = isStageControlFlow(props.stage);

  const hideStage = useHideStageIfInPromotionMode(props.stage);

  const queryFreightMutation = useMutation(queryFreight);

  const totalSubscribersToThisStage = dictionaryContext?.subscribersByStage?.[stageName]?.size || 0;

  const manualApproveActionMutation = useMutation(approveFreight, {
    onSuccess: (_, vars) => {
      message.success(
        `Freight ${actionContext?.action?.freight?.alias} has been manually approved for stage ${vars.stage}`
      );

      actionContext?.cancel();
    }
  });

  const dropdownItems = useGetPromotionDropdownItems(props.stage);

  let descriptionItems: ReactNode;

  const lastPromotion = getLastPromotionDate(props.stage);
  const date = timestampDate(lastPromotion) as Date;

  if (!controlFlow) {
    descriptionItems = (
      <Flex className='text-[10px]' gap={8} wrap vertical>
        <Flex gap={24} justify='center'>
          <StageNodePhase stage={props.stage} />
          {stageHealth?.status && (
            <Flex align='center' gap={4}>
              {stageHealth?.status}
              <HealthStatusIcon noTooltip className='text-[8px]' health={stageHealth} />
            </Flex>
          )}
        </Flex>

        <center>
          <PullRequestLink stage={props.stage} />

          {dictionaryContext?.hasAnalysisRunLogsUrlTemplate && (
            <AnalysisRunLogsLink stage={props.stage} />
          )}
        </center>
      </Flex>
    );
  }

  const { isOver, setNodeRef } = useDroppable({
    id: props.stage.metadata?.name || 'stage-node',
    data: {
      requestedFreightNames: props.stage.spec?.requestedFreight.map((f) => f.origin?.name) || []
    }
  });

  return (
    <div
      ref={setNodeRef}
      style={{
        transform: isOver ? 'scale(0.98)' : undefined,
        transition: 'transform 0.1s ease, opacity 0.2s ease'
      }}
    >
      <Card
        styles={{
          header: {
            ...headerStyle,
            textShadow: '1px 1px 2px rgba(0, 0, 0, 0.15)',
            paddingRight: '8px'
          },
          body: {
            height: '100%',
            position: 'relative'
          }
        }}
        title={
          <>
            {autoPromotionMode && (
              <FontAwesomeIcon title='Auto Promotion' icon={faBolt} className='text-[10px] mr-1' />
            )}
            <span className='text-xs text-wrap mr-auto'>{props.stage.metadata?.name}</span>
          </>
        }
        extra={
          <Space size={6}>
            <ArgoCDLink
              stage={props.stage}
              buttonProps={{
                size: 'small',
                icon: <img src='/argo-logo.svg' alt='ArgoCD' style={{ width: '18px' }} />
              }}
            />
            <Dropdown
              trigger={['click']}
              overlayClassName='w-fit'
              menu={{
                items: dropdownItems
              }}
            >
              <Button
                size='small'
                loading={queryFreightMutation.isPending}
                icon={<FontAwesomeIcon icon={faTruckArrowRight} size='sm' />}
              />
            </Dropdown>
            <Button
              icon={<FontAwesomeIcon icon={faBarsStaggered} className='mt-1' />}
              size='small'
              onClick={() =>
                navigate(
                  generatePath(paths.stage, {
                    name: props.stage?.metadata?.namespace,
                    stageName: props.stage?.metadata?.name
                  })
                )
              }
            />
          </Space>
        }
        className={classNames(
          'stage-node',
          style['stage-node-size'],
          {
            'opacity-40': hideStage
          },
          'postiion-relative'
        )}
        size='small'
        variant='borderless'
      >
        <DropOverlay isOver={isOver} stage={props.stage} />
        {controlFlow && (
          <Typography.Text type='secondary'>
            <FontAwesomeIcon icon={faTruckArrowRight} className='mr-2' />
            Control Flow
          </Typography.Text>
        )}

        {descriptionItems}

        <div className='my-2'>
          {actionContext?.action?.type === IAction.MANUALLY_APPROVE ? (
            <Button
              className='success'
              size='small'
              loading={manualApproveActionMutation.isPending}
              onClick={() => {
                manualApproveActionMutation.mutate({
                  stage: props.stage?.metadata?.name || '',
                  project: props.stage?.metadata?.namespace,
                  name: actionContext?.action?.freight?.metadata?.name
                });
              }}
            >
              Approve
            </Button>
          ) : (
            <StageFreight stage={props.stage} />
          )}
        </div>

        {lastPromotion && (
          <Link
            to={generatePath(paths.promotion, {
              name: props.stage?.metadata?.namespace,
              promotionId: props.stage?.status?.lastPromotion?.name
            })}
          >
            <Flex gap={4} align='center' justify='center' className='text-[10px]'>
              <span>Last Promotion: </span>
              <span title={date?.toString()}>
                {formatDistance(date, new Date(), { addSuffix: true })}
              </span>
              <FontAwesomeIcon icon={faExternalLink} className='text-[6px]' />
            </Flex>
          </Link>
        )}
      </Card>

      {!graphContext?.stackedNodesParents?.includes(stageNodeIndex) &&
        totalSubscribersToThisStage > 0 && (
          <Button
            style={{ width: 16, height: 16 }}
            icon={<FontAwesomeIcon icon={faMinus} />}
            size='small'
            className='absolute top-[50%] right-0 translate-x-[50%] translate-y-[-50%] text-[8px] z-10'
            onClick={() => graphContext?.onStack(stageNodeIndex)}
          />
        )}
    </div>
  );
};
