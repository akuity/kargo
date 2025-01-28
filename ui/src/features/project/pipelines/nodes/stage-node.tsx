import { useMutation } from '@connectrpc/connect-query';
import {
  IconDefinition,
  faArrowRight,
  faBullseye,
  faCircle,
  faCircleNotch,
  faCodePullRequest,
  faExclamationTriangle,
  faGear,
  faRobot,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, message, Tooltip } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { ReactNode } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { getCurrentFreight, selectFreightByWarehouse } from '@ui/features/common/utils';
import { willStagePromotionOpenPR } from '@ui/features/promotion-directives/utils';
import { getColors } from '@ui/features/stage/utils';
import {
  approveFreight,
  promoteToStage
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { usePipelineContext } from '../context/use-pipeline-context';
import { FreightTimelineAction, NodeDimensions } from '../types';
import { isPromoting, PipelineStateHook } from '../utils/state';
import { isStageControlFlow, onError } from '../utils/util';

import styles from './custom-node.module.less';
import { FreightIndicators } from './freight-indicators';
import { FreightLabel } from './freight-label';
import { lastVerificationErrored } from './util';

export const StageNodeDimensions = () =>
  ({
    // MUST BE SAME AS DEFINED IN custom-node.module.less .stageNode
    width: 250,
    height: 198
  }) as NodeDimensions;

type StageNodeProps = {
  stage: Stage;
};

export const StageNode = (props: StageNodeProps) => {
  const navigate = useNavigate();

  const pipelineContext = usePipelineContext();

  const colorMap = getColors(pipelineContext?.project || '', [props.stage]);

  const stageColor = colorMap[props.stage?.metadata?.name || ''];

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
        'opacity-50': isFaded()
      })}
      onMouseEnter={() => pipelineContext?.onHover(true, props.stage?.metadata?.name || '', true)}
      onMouseLeave={() => pipelineContext?.onHover(false, props.stage?.metadata?.name || '', true)}
      onClick={onClick}
    >
      <div
        className={classNames(styles.header, 'text-white text-base')}
        style={{ backgroundColor: stageColor }}
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

const Nodule = (props: {
  begin?: boolean;
  nodeHeight: number;
  onClick?: () => void;
  selected?: boolean;
  icon?: IconDefinition;
}) => {
  const noduleHeight = 30;
  const top = -15;
  return (
    <Tooltip
      title={
        props.icon ? '' : props.begin ? 'Promote into Stage' : 'Promote to downstream Subscribers'
      }
    >
      <div
        onClick={(e) => {
          e.stopPropagation();
          if (props.onClick) {
            props.onClick();
          }
        }}
        style={{
          top: top,
          height: noduleHeight,
          width: noduleHeight,
          left: props.begin ? -noduleHeight / 2 : 'auto',
          right: props.begin ? 'auto' : -noduleHeight / 2
        }}
        className={`cursor-pointer select-none z-10 flex items-center justify-center hover:text-white border border-sky-300 border-solid hover:bg-blue-400 absolute rounded-lg 
          ${props.selected ? 'text-white bg-blue-400' : 'bg-white text-blue-500'}`}
      >
        <FontAwesomeIcon
          icon={props.icon ? props.icon : props.begin ? faBullseye : faTruckArrowRight}
        />
      </div>
    </Tooltip>
  );
};
