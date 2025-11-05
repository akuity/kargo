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
import { ArgoCDLink } from '@ui/features/project/pipelines/nodes/argocd-link';
import {
  approveFreight,
  promoteToStage,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { useDictionaryContext } from '../context/dictionary-context';
import { useGraphContext } from '../context/graph-context';
import { stageIndexer } from '../graph/node-indexer';
import { DropOverlay } from '../promotion/drag-and-drop/drop-overlay';
import { useManualApprovalModal } from '../promotion/use-manual-approval-modal';

import { AnalysisRunLogsLink } from './analysis-run-logs-link';
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
  const projectName = props.stage?.metadata?.namespace || '';
  const stageName = props.stage?.metadata?.name || '';

  const navigate = useNavigate();
  const dictionaryContext = useDictionaryContext();
  const graphContext = useGraphContext();
  const actionContext = useActionContext();

  const stageNodeIndex = useMemo(() => stageIndexer.index(props.stage), [props.stage]);

  const headerStyle = useStageHeaderStyle(props.stage);

  const autoPromotionMode = dictionaryContext?.stageAutoPromotionMap?.[stageName];

  const stagePhase = getStagePhase(props.stage);
  const stageHealth = getStageHealth(props.stage);

  const controlFlow = isStageControlFlow(props.stage);

  const hideStage = useHideStageIfInPromotionMode(props.stage);

  const upstreamFreights = useGetUpstreamFreight(props.stage);

  const promoteActionMutation = useMutation(promoteToStage, {
    onSuccess: (response) => {
      navigate(
        generatePath(paths.promotion, {
          name: projectName,
          promotionId: response.promotion?.metadata?.name
        })
      );
    }
  });

  const queryFreightMutation = useMutation(queryFreight);

  const checkFreightPromotionEligibility = async (freight: string) => {
    const freightResponse = await queryFreightMutation.mutateAsync({
      project: projectName,
      stage: stageName
    });

    return Boolean(
      freightResponse?.groups?.['']?.freight?.find((item) => item?.metadata?.name === freight)
    );
  };

  const showManualApproveModal = useManualApprovalModal();

  const ensureEligibilityBeforeAction = async (
    freight: string | undefined,
    onEligible: (eligibleFreight: string) => void,
    options?: { runAfterApproval?: (eligibleFreight: string) => void }
  ) => {
    if (!freight) {
      return;
    }

    const isEligible = await checkFreightPromotionEligibility(freight);

    if (!isEligible) {
      showManualApproveModal({
        freight,
        projectName,
        stage: stageName,
        onApprove: options?.runAfterApproval
          ? () => {
              options.runAfterApproval?.(freight);
            }
          : undefined
      });
      return;
    }

    onEligible(freight);
  };

  const totalSubscribersToThisStage = dictionaryContext?.subscribersByStage?.[stageName]?.size || 0;

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
            name: projectName,
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

  const navigateToPromote = (freight: string) =>
    navigate(
      generatePath(paths.promote, {
        name: projectName,
        freight,
        stage: stageName
      })
    );

  const promoteFreight = (freight: string) =>
    promoteActionMutation.mutate({
      stage: stageName,
      project: projectName,
      freight
    });

  const handleNavigateWithEligibility = (freight?: string) =>
    ensureEligibilityBeforeAction(freight, (eligibleFreight) => navigateToPromote(eligibleFreight));

  const handleInstantPromoteWithEligibility = (freight?: string) =>
    ensureEligibilityBeforeAction(freight, (eligibleFreight) => promoteFreight(eligibleFreight), {
      runAfterApproval: (eligibleFreight) => promoteFreight(eligibleFreight)
    });

  const buildFreightMenuItems = (
    handler: (freight?: string) => Promise<void> | void
  ): ItemType[] | undefined =>
    upstreamFreights?.map((upstreamFreight) => ({
      key: upstreamFreight?.name,
      label: upstreamFreight?.origin?.name,
      onClick: () => {
        void handler(upstreamFreight?.name);
      }
    }));

  const getSingleFreightHandler = (handler: (freight?: string) => Promise<void> | void) =>
    upstreamFreights?.length === 1
      ? () => {
          void handler(upstreamFreights?.[0]?.name);
        }
      : undefined;

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

  const hasUpstreamFreights = (upstreamFreights?.length || 0) > 0;
  const hasMultipleUpstreamFreights = (upstreamFreights?.length || 0) > 1;

  if (hasUpstreamFreights) {
    dropdownItems.push({
      key: 'upstream-freight-promo',
      label: 'Promote from upstream',
      onClick: getSingleFreightHandler(handleNavigateWithEligibility),
      children: hasMultipleUpstreamFreights
        ? buildFreightMenuItems(handleNavigateWithEligibility)
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
      onClick: getSingleFreightHandler(handleInstantPromoteWithEligibility),
      children: hasMultipleUpstreamFreights
        ? buildFreightMenuItems(handleInstantPromoteWithEligibility)
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
