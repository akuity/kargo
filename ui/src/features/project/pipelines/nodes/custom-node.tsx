import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import {
  faAnchor,
  faBuilding,
  faCircleNotch,
  faCodePullRequest,
  faExclamationTriangle,
  faGear,
  faQuestion,
  faRefresh,
  faRobot
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Handle, Position } from '@xyflow/react';
import { Button, Tooltip } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { PropsWithChildren, useContext } from 'react';

import { ColorContext } from '@ui/context/colors';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { getCurrentFreight, selectFreightByWarehouse } from '@ui/features/common/utils';
import { willStagePromotionOpenPR } from '@ui/features/promotion-directives/utils';
import { ColorMapHex, parseColorAnnotation } from '@ui/features/stage/utils';
import { Freight, RepoSubscription, Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { usePipelineContext } from '../context/use-pipeline-context';

import styles from './custom-node.module.less';
import { FreightIndicators } from './freight-indicators';
import { FreightLabel } from './freight-label';
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

  const warehouseName = props.warehouse?.metadata?.name || '';

  let refreshing = false;

  for (const condition of props.warehouse?.status?.conditions || []) {
    if (condition.type === 'Reconciling' && condition.status === 'True') {
      refreshing = true;
    }
  }

  return (
    <div className={classNames(styles.warehouseNode)}>
      <div className={classNames(styles.header)}>
        <h3>{warehouseName}</h3>

        <div className='ml-auto space-x-2'>
          {refreshing && <FontAwesomeIcon icon={faCircleNotch} spin />}
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
        <Button icon={<FontAwesomeIcon icon={faRefresh} />} size='small' className='mx-auto'>
          Refresh
        </Button>
      </div>
    </div>
  );
};

CustomNode.StageNode = (props: { stage: Stage }) => {
  const { warehouseColorMap } = useContext(ColorContext);

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

  return (
    <div className={classNames(styles.stageNode)}>
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

      <div className={classNames(styles.body)}>
        {/* TODO(Marvin9): approval action */}
        <div className='text-sm flex flex-col items-center justify-center'>
          <FreightIndicators
            freight={currentFreight}
            selectedFreight={visibleFreight}
            onClick={(idx) => setVisibleFreight(idx)}
          />

          <FreightLabel freight={currentFreight[visibleFreight]} />
        </div>
      </div>

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
    </div>
  );
};
