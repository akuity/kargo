import { useMutation } from '@connectrpc/connect-query';
import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import {
  faAnchor,
  faArrowRight,
  faBuilding,
  faCircle,
  faCircleNotch,
  faCodePullRequest,
  faExclamationCircle,
  faExclamationTriangle,
  faGear,
  faQuestion,
  faRefresh,
  faRobot
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Handle, Position } from '@xyflow/react';
import { Button, message, Tooltip } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { PropsWithChildren, ReactNode, useContext } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { getCurrentFreight, selectFreightByWarehouse } from '@ui/features/common/utils';
import { willStagePromotionOpenPR } from '@ui/features/promotion-directives/utils';
import { ColorMapHex, parseColorAnnotation } from '@ui/features/stage/utils';
import {
  approveFreight,
  promoteToStage,
  refreshWarehouse
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, RepoSubscription, Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { usePipelineContext } from '../context/use-pipeline-context';
import { MessageTooltip } from '../message-tooltip';
import { FreightTimelineAction } from '../types';
import { isPromoting, PipelineStateHook } from '../utils/state';
import { isStageControlFlow, onError } from '../utils/util';

import styles from './custom-node.module.less';
import { FreightIndicators } from './freight-indicators';
import { FreightLabel } from './freight-label';
import { Nodule } from './stage-node';
import { lastVerificationErrored } from './util';

export const CustomNode = ({
  data
}: {
  data: {
    label: string;
    value: Warehouse | RepoSubscription | Stage;
    freightMap: { [key: string]: Freight };
  };
}) => {
  // todo: why there'd be no data.value?
  if (!data.value) {
    return null;
  }

  if (data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Warehouse') {
    return (
      <CustomNode.Container>
        <CustomNode.WarehouseNode warehouse={data.value} />
      </CustomNode.Container>
    );
  }

  if (data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription') {
    return (
      <CustomNode.Container>
        <CustomNode.SubscriptionNode subscription={data.value} />
      </CustomNode.Container>
    );
  }

  if (data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Stage') {
    return (
      <CustomNode.Container>
        <CustomNode.StageNode stage={data.value} />
      </CustomNode.Container>
    );
  }

  return <CustomNode.Container>Unknown Node</CustomNode.Container>;
};

CustomNode.Container = (props: PropsWithChildren<object>) => (
  <>
    <Handle type='target' position={Position.Left} />
    <div className={styles.container}>{props.children}</div>
    <Handle type='source' position={Position.Right} />
  </>
);

CustomNode.SubscriptionNode = (props: { subscription: RepoSubscription }) => {
  let icon: IconProp = faQuestion;

  if (props.subscription?.chart) {
    icon = faAnchor;
  } else if (props.subscription?.git) {
    icon = faGitAlt;
  } else if (props.subscription?.image) {
    icon = faDocker;
  }

  const url =
    props.subscription?.git?.repoURL ||
    props.subscription?.image?.repoURL ||
    props.subscription?.chart?.repoURL;

  return (
    <div className={classNames(styles.repoSubscriptionNode)}>
      <div className={classNames(styles.header)}>
        <h3>Subscription</h3>

        <FontAwesomeIcon className='ml-auto text-base' icon={icon} />
      </div>

      <div className={classNames(styles.body)}>
        <Tooltip title={url}>
          <span className='block w-36 overflow-hidden text-ellipsis whitespace-nowrap'>{url}</span>
        </Tooltip>
      </div>
    </div>
  );
};

CustomNode.WarehouseNode = (props: { warehouse: Warehouse }) => {
  const { warehouseColorMap } = useContext(ColorContext);

  const pipelineContext = usePipelineContext();

  const navigate = useNavigate();

  const refeshWarehouseMutation = useMutation(refreshWarehouse, {
    onError,
    onSuccess: () => {
      message.success('Warehouse successfully refreshed');
      pipelineContext?.state.clear();
      // TODO: refetchFreightData
    }
  });

  const warehouseName = props.warehouse?.metadata?.name || '';

  let refreshing = false;

  for (const condition of props.warehouse?.status?.conditions || []) {
    if (condition.type === 'Reconciling' && condition.status === 'True') {
      refreshing = true;
    }
  }

  let hasError = false;
  let notReady = false;
  let hasReconcilingCondition = false;
  let errMessage = '';

  for (const condition of props.warehouse?.status?.conditions || []) {
    if (condition.type === 'Healthy' && condition.status === 'False') {
      hasError = true;
      errMessage = condition.message || '';
    }

    if (condition.type === 'Reconciling') {
      hasReconcilingCondition = true;
    }

    if (condition.type === 'Ready' && condition.status === 'False') {
      notReady = true;
      errMessage = condition.message || '';
    }
  }

  if (notReady && !hasReconcilingCondition) {
    hasError = true;
  }

  return (
    <div
      className={classNames(styles.warehouseNode)}
      onClick={() =>
        navigate(
          generatePath(paths.warehouse, {
            name: pipelineContext?.project,
            warehouseName: props.warehouse?.metadata?.name
          })
        )
      }
    >
      <div className={classNames(styles.header)}>
        <h3>{warehouseName}</h3>

        <div className='ml-auto space-x-2'>
          {refreshing && <FontAwesomeIcon icon={faCircleNotch} spin />}
          {/* TODO: make this error tooltip beautiful */}
          {hasError && (
            <MessageTooltip
              message={
                <div className='flex flex-col gap-4 overflow-y-scroll text-wrap max-h-48'>
                  <div
                    className='cursor-pointer min-w-0'
                    onClick={() => {
                      if (message) {
                        navigator.clipboard.writeText(errMessage);
                      }
                    }}
                  >
                    <div className='flex text-wrap'>
                      <FontAwesomeIcon icon={faExclamationCircle} className='mr-2 mt-1 pl-1' />
                      {errMessage}
                    </div>
                  </div>
                </div>
              }
              icon={faExclamationCircle}
              iconClassName='text-red-500'
            />
          )}
          <FontAwesomeIcon
            icon={faBuilding}
            className='text-base'
            style={{
              color: warehouseColorMap[warehouseName]
            }}
          />
        </div>
      </div>

      <div className={classNames(styles.body, 'flex')}>
        <Button
          icon={<FontAwesomeIcon icon={faRefresh} />}
          size='small'
          className='mx-auto'
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            refeshWarehouseMutation.mutate({
              name: props.warehouse?.metadata?.name,
              project: pipelineContext?.project
            });
          }}
        >
          Refresh
        </Button>
      </div>
    </div>
  );
};

CustomNode.StageNode = (props: { stage: Stage }) => {
  const { warehouseColorMap } = useContext(ColorContext);

  const navigate = useNavigate();

  const stageColor =
    parseColorAnnotation(props.stage) ||
    warehouseColorMap[props.stage?.spec?.requestedFreight?.[0]?.origin?.name || ''];

  const pipelineContext = usePipelineContext();

  const currentFreight = getCurrentFreight(props.stage)
    ?.map((freight) => pipelineContext?.fullFreightById[freight?.name || ''])
    .filter(Boolean) as Freight[];

  const hasNoSubscribers =
    (pipelineContext?.subscribersByStage[props.stage?.metadata?.name || '']?.size || 0) <= 1;

  const [_visibleFreight, setVisibleFreight] = useLocalStorage(
    `${pipelineContext?.project}-${props.stage?.metadata?.name}`,
    selectFreightByWarehouse(currentFreight, pipelineContext?.selectedWarehouse || '')
  );

  let visibleFreight = _visibleFreight;
  if (pipelineContext?.selectedWarehouse) {
    visibleFreight = selectFreightByWarehouse(currentFreight, pipelineContext.selectedWarehouse);
  }

  const manualApproveActionMutation = useMutation(approveFreight, {
    onError,
    onSuccess: () => {
      message.success(`Freight ${pipelineContext?.state?.freight} has been manually approved.`);
      // TODO: refetchFreightData
      pipelineContext?.state?.clear();
    }
  });

  const promoteActionMutation = useMutation(promoteToStage, {
    onError,
    onSuccess: () => {
      message.success(
        `Promotion request for stage ${pipelineContext?.state?.stage} has been successfully submitted.`
      );
      pipelineContext?.state?.clear();
    }
  });

  let onClick = () => {
    navigate(
      generatePath(paths.stage, {
        name: pipelineContext?.project,
        stageName: props.stage?.metadata?.name || ''
      })
    );
  };

  if (pipelineContext?.state?.action === FreightTimelineAction.ManualApproval) {
    onClick = () => {
      manualApproveActionMutation.mutate({
        stage: props.stage?.metadata?.name || '',
        project: pipelineContext?.project,
        name: pipelineContext?.state?.freight
      });
    };
  }

  if (pipelineContext?.state?.action === FreightTimelineAction.PromoteFreight) {
    onClick = () => {
      pipelineContext?.state?.setStage(props.stage?.metadata?.name || '');
      promoteActionMutation.mutate({
        stage: props.stage?.metadata?.name || '',
        project: pipelineContext?.project,
        freight: pipelineContext?.state?.freight
      });
    };
  }

  let Body: ReactNode;

  if (
    pipelineContext?.state?.action === FreightTimelineAction.ManualApproval ||
    pipelineContext?.state?.action === FreightTimelineAction.PromoteFreight
  ) {
    Body = (
      <div className='text-sm flex flex-col items-center justify-center py-5'>
        <Button
          icon={
            <FontAwesomeIcon
              icon={
                pipelineContext?.state?.action === FreightTimelineAction.ManualApproval
                  ? faCircle
                  : faArrowRight
              }
            />
          }
          disabled={isStageControlFlow(props.stage)}
          className='uppercase'
        >
          {pipelineContext?.state?.action === FreightTimelineAction.ManualApproval
            ? 'Approve'
            : 'Promote'}
        </Button>
      </div>
    );
  } else {
    Body = (
      <div className='text-sm flex flex-col items-center justify-center'>
        <FreightIndicators
          freight={currentFreight}
          selectedFreight={visibleFreight}
          onClick={(idx) => setVisibleFreight(idx)}
        />

        <FreightLabel freight={currentFreight[visibleFreight]} />
      </div>
    );
  }

  let Promoters: ReactNode;

  if (
    pipelineContext?.state?.action !== FreightTimelineAction.ManualApproval &&
    pipelineContext?.state?.action !== FreightTimelineAction.PromoteFreight
  ) {
    Promoters = (
      <>
        {!isStageControlFlow(props.stage) && (
          <div className='absolute top-[50%]'>
            <Nodule
              begin
              nodeHeight={20}
              onClick={() =>
                pipelineContext?.onPromoteClick(props.stage, FreightTimelineAction.Promote)
              }
              selected={pipelineContext?.state?.action === FreightTimelineAction.Promote}
            />
          </div>
        )}
        {!hasNoSubscribers && (
          <div className='absolute top-[50%] right-0'>
            <Nodule
              nodeHeight={20}
              onClick={() =>
                pipelineContext?.onPromoteClick(
                  props.stage,
                  FreightTimelineAction.PromoteSubscribers
                )
              }
              selected={pipelineContext?.state?.action === FreightTimelineAction.PromoteSubscribers}
            />
          </div>
        )}
      </>
    );
  }

  const isFaded = () => {
    if (!isPromoting(pipelineContext?.state as PipelineStateHook)) {
      return false;
    }

    if (pipelineContext?.state?.action === FreightTimelineAction.Promote) {
      return pipelineContext?.state?.stage !== props.stage?.metadata?.name;
    }

    if (pipelineContext?.state?.action === FreightTimelineAction.PromoteSubscribers) {
      return (
        !props.stage?.metadata?.name ||
        !pipelineContext?.subscribersByStage[pipelineContext?.state?.stage || '']?.has(
          props.stage?.metadata?.name
        )
      );
    }

    return false;
  };

  return (
    <div
      className={classNames(styles.stageNode, {
        [styles.highlightedNode]:
          pipelineContext?.highlightedStages?.[props.stage?.metadata?.name || ''],
        'opacity-70': isFaded()
      })}
      onMouseEnter={() => pipelineContext?.onHover(true, props.stage?.metadata?.name || '', true)}
      onMouseLeave={() => pipelineContext?.onHover(false, props.stage?.metadata?.name || '', true)}
      onClick={onClick}
    >
      <div
        className={classNames(styles.header, 'text-white text-base')}
        style={{ backgroundColor: ColorMapHex[stageColor] }}
      >
        <h3>{props.stage?.metadata?.name}</h3>

        {/* TODO(Marvin9): Make plugin hole here */}
        <div className='ml-auto space-x-1'>
          {willStagePromotionOpenPR(props.stage) && (
            <Tooltip title='contains git-open-pr'>
              <FontAwesomeIcon icon={faCodePullRequest} />
            </Tooltip>
          )}

          {pipelineContext?.autoPromotionMap[props.stage?.metadata?.name || ''] && (
            <Tooltip title='Auto Promotion Enabled'>
              <FontAwesomeIcon icon={faRobot} />
            </Tooltip>
          )}

          {!props.stage?.status?.currentPromotion && props.stage?.status?.lastPromotion && (
            <PromotionStatusIcon
              placement='top'
              status={props.stage?.status?.lastPromotion?.status}
              color='white'
            />
          )}

          {props.stage.status?.currentPromotion ? (
            <Tooltip
              title={`Freight ${props.stage.status?.currentPromotion.freight?.name} is being promoted`}
            >
              <FontAwesomeIcon icon={faGear} spin={true} />
            </Tooltip>
          ) : props.stage.status?.phase === 'Verifying' ? (
            <Tooltip title='Verifying Current Freight'>
              <FontAwesomeIcon icon={faCircleNotch} spin={true} />
            </Tooltip>
          ) : (
            props.stage.status?.health && (
              <HealthStatusIcon health={props.stage.status?.health} hideColor={true} />
            )
          )}

          {lastVerificationErrored(props.stage) && (
            <Tooltip
              title={
                <>
                  <div>
                    <b>Verification Failure:</b>
                  </div>
                  {props.stage?.status?.freightHistory?.[0]?.verificationHistory?.[0]?.message}
                </>
              }
            >
              <FontAwesomeIcon icon={faExclamationTriangle} />
            </Tooltip>
          )}
        </div>
      </div>

      <div className={classNames(styles.body)}>{Body}</div>

      {props.stage?.status?.lastPromotion?.finishedAt && (
        <div className={classNames(styles.footer)}>
          <span className='uppercase text-xs text-gray-400'>Last Promo: </span>
          <b>
            {formatDistance(
              timestampDate(props.stage.status.lastPromotion.finishedAt) as Date,
              new Date(),
              {
                addSuffix: true
              }
            )}
          </b>
        </div>
      )}

      {Promoters}
    </div>
  );
};
