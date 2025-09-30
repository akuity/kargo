import { useMutation } from '@connectrpc/connect-query';
import { useDroppable } from '@dnd-kit/core';
import {
  faBarsStaggered,
  faBolt,
  faBoltLightning,
  faCircleNotch,
  faExternalLink,
  faMinus,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Dropdown, Flex, message, Space, Typography } from 'antd';
import { ItemType } from 'antd/es/menu/interface';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { ReactNode, useMemo } from 'react';
import { generatePath, Link, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import {
  approveFreight,
  promoteToStage
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { useDictionaryContext } from '../context/dictionary-context';
import { useGraphContext } from '../context/graph-context';
import { stageIndexer } from '../graph/node-indexer';
import { DropOverlay } from '../promotion/drag-and-drop/drop-overlay';

import { AnalysisRunLogsLink } from './analysis-run-logs-link';
import { ArgoCDLink } from './argocd-link';
import style from './node-size-source-of-truth.module.less';
import { PullRequestLink } from './pull-request-link';
import { StageFreight } from './stage-freight';
import {
  getLastPromotionDate,
  getStageHealth,
  isStageControlFlow,
  useHideStageIfInPromotionMode,
  useStageHeaderStyle
} from './stage-meta-utils';
import { useGetUpstreamFreight } from './use-get-upstream-freight';

import './stage-node.less';

export const StageNode = (props: { stage: Stage }) => {
  const navigate = useNavigate();
  const dictionaryContext = useDictionaryContext();
  const graphContext = useGraphContext();
  const actionContext = useActionContext();

  const stageNodeIndex = useMemo(() => stageIndexer.index(props.stage), [props.stage]);

  const headerStyle = useStageHeaderStyle(props.stage);

  const autoPromotionMode =
    dictionaryContext?.stageAutoPromotionMap?.[props.stage?.metadata?.name || ''];

  const stagePhase = getStagePhase(props.stage);
  const stageHealth = getStageHealth(props.stage);

  const controlFlow = isStageControlFlow(props.stage);

  const hideStage = useHideStageIfInPromotionMode(props.stage);

  const upstreamFreights = useGetUpstreamFreight(props.stage);

  const promoteActionMutation = useMutation(promoteToStage, {
    onSuccess: (response) => {
      navigate(
        generatePath(paths.promotion, {
          name: props.stage?.metadata?.namespace,
          promotionId: response.promotion?.metadata?.name
        })
      );
    }
  });

  const totalSubscribersToThisStage =
    dictionaryContext?.subscribersByStage?.[props.stage?.metadata?.name || '']?.size || 0;

  const manualApproveActionMutation = useMutation(approveFreight, {
    onSuccess: (_, vars) => {
      message.success(
        `Freight ${actionContext?.action?.freight?.alias} has been manually approved for stage ${vars.stage}`
      );

      actionContext?.cancel();
    }
  });

  let descriptionItems: ReactNode;

  const lastPromotion = getLastPromotionDate(props.stage);
  const date = timestampDate(lastPromotion) as Date;

  if (!controlFlow) {
    let Phase = (
      <Flex align='center' gap={4}>
        {stagePhase}{' '}
        <StageConditionIcon
          className='text-[10px]'
          conditions={props.stage?.status?.conditions || []}
          noTooltip
        />
        {stagePhase === 'Promoting' && (
          <FontAwesomeIcon icon={faExternalLink} className='text-[8px]' />
        )}
      </Flex>
    );

    if (stagePhase === 'Promoting') {
      Phase = (
        <Link
          to={generatePath(paths.promotion, {
            name: props.stage?.metadata?.namespace,
            promotionId: props.stage?.status?.currentPromotion?.name || ''
          })}
        >
          {Phase}
        </Link>
      );
    }

    descriptionItems = (
      <Flex className='text-[10px]' gap={8} wrap vertical>
        <Flex gap={24} justify='center'>
          {Phase}
          {stageHealth?.status && (
            <Flex gap={16}>
              <Flex align='center' gap={4}>
                {stageHealth?.status}
                <HealthStatusIcon noTooltip className='text-[8px]' health={stageHealth} />
              </Flex>

              <ArgoCDLink stage={props.stage} shards={dictionaryContext?.argocdShards} />
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

  const dropdownItems: ItemType[] = [];

  if (!controlFlow) {
    dropdownItems.push({
      key: 'promote',
      label: 'Promote',
      onClick: () => actionContext?.actPromote(IAction.PROMOTE, props.stage)
    });
  }

  if (controlFlow || totalSubscribersToThisStage > 1) {
    dropdownItems.push({
      key: 'promote-downstream',
      label: 'Promote to downstream',
      onClick: () => actionContext?.actPromote(IAction.PROMOTE_DOWNSTREAM, props.stage)
    });
  }

  if ((upstreamFreights?.length || 0) > 0) {
    dropdownItems.push({
      key: 'upstream-freight-promo',
      label: 'Promote from upstream',
      onClick:
        upstreamFreights?.length === 1
          ? () =>
              navigate(
                generatePath(paths.promote, {
                  name: props.stage?.metadata?.namespace,
                  freight: upstreamFreights?.[0]?.name,
                  stage: props.stage?.metadata?.name
                })
              )
          : undefined,
      children:
        (upstreamFreights?.length || 0) > 1
          ? upstreamFreights?.map((f) => ({
              key: f?.name,
              label: f?.origin?.name,
              onClick: () =>
                navigate(
                  generatePath(paths.promote, {
                    name: props.stage?.metadata?.namespace,
                    freight: f?.name,
                    stage: props.stage?.metadata?.name
                  })
                )
            }))
          : undefined
    });

    dropdownItems.push({
      key: 'quick-promote-upstream-freight-promo',
      label: (
        <>
          {promoteActionMutation.isPending ? (
            <FontAwesomeIcon icon={faCircleNotch} className='mr-1' spin />
          ) : (
            <Typography.Text type='danger' className='mr-2'>
              <FontAwesomeIcon icon={faBoltLightning} />
            </Typography.Text>
          )}
          Instant promote from upstream
        </>
      ),
      onClick:
        upstreamFreights?.length === 1
          ? () =>
              promoteActionMutation.mutate({
                stage: props.stage?.metadata?.name,
                project: props.stage?.metadata?.namespace,
                freight: upstreamFreights?.[0]?.name
              })
          : undefined,
      children:
        (upstreamFreights?.length || 0) > 1
          ? upstreamFreights?.map((f) => ({
              key: f?.name,
              label: f?.origin?.name,
              onClick: () =>
                promoteActionMutation.mutate({
                  stage: props.stage?.metadata?.name,
                  project: props.stage?.metadata?.namespace,
                  freight: f?.name
                })
            }))
          : undefined
    });
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
            textShadow: '1px 1px 2px rgba(0, 0, 0, 0.15)'
          },
          body: {
            height: '100%',
            position: 'relative'
          }
        }}
        title={
          <Flex align='center'>
            {autoPromotionMode && (
              <FontAwesomeIcon title='Auto Promotion' icon={faBolt} className='text-[10px] mr-1' />
            )}
            <span className='text-xs text-wrap mr-auto'>{props.stage.metadata?.name}</span>
            <Space>
              <Dropdown
                trigger={['hover']}
                overlayClassName='w-fit'
                menu={{
                  items: dropdownItems
                }}
              >
                <Button
                  size='small'
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
          </Flex>
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
