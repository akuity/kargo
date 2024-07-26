import {
  IconDefinition,
  faArrowRight,
  faBullseye,
  faCircleCheck,
  faCircleNotch,
  faCodePullRequest,
  faGear,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';
import { useState } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { FreightTimelineAction, NodeDimensions } from '../types';

import { FreightIndicators } from './freight-indicators';
import { FreightLabel } from './freight-label';
import { StageNodeFooter } from './stage-node-footer';
import * as styles from './stage-node.module.less';

export const StageNodeDimensions = () =>
  ({
    width: 215,
    height: 165
  }) as NodeDimensions;

export const StageNode = ({
  stage,
  color,
  height,
  faded,
  onPromoteClick,
  projectName,
  hasNoSubscribers,
  action,
  currentFreight,
  onClick,
  onHover,
  highlighted
}: {
  stage: Stage;
  color: string;
  height: number;
  faded: boolean;
  projectName?: string;
  hasNoSubscribers?: boolean;
  action?: FreightTimelineAction;
  onPromoteClick: (type: FreightTimelineAction) => void;
  currentFreight: Freight[];
  onClick?: () => void;
  onHover: (hovering: boolean) => void;
  highlighted?: boolean;
}) => {
  const navigate = useNavigate();
  const [visibleFreight, setVisibleFreight] = useState(0);

  return (
    <>
      <div
        className={`${styles.node} ${faded ? styles.faded : ''} ${
          highlighted ? styles.highlighted : ''
        }`}
        style={{
          backgroundColor: color,
          borderColor: color,
          position: 'relative',
          cursor: 'pointer'
        }}
        onClick={() => {
          if (onClick) {
            onClick();
          } else {
            navigate(
              generatePath(paths.stage, { name: projectName, stageName: stage.metadata?.name })
            );
          }
        }}
        onMouseEnter={() => onHover(true)}
        onMouseLeave={() => onHover(false)}
      >
        <h3>
          <div className='truncate pb-1 mr-auto'>{stage.metadata?.name}</div>
          <div className='flex gap-1'>
            {(stage?.spec?.promotionMechanisms?.gitRepoUpdates || []).some(
              (g) => g.pullRequest
            ) && (
              <Tooltip title='PR Promotion Enabled'>
                <FontAwesomeIcon icon={faCodePullRequest} />
              </Tooltip>
            )}
            {!stage?.status?.currentPromotion && stage.status?.lastPromotion && (
              <div className='pb-1'>
                <PromotionStatusIcon
                  placement='top'
                  status={stage.status?.lastPromotion.status}
                  color='white'
                  size='1x'
                />
              </div>
            )}
            {stage.status?.currentPromotion ? (
              <Tooltip
                title={`Freight ${stage.status?.currentPromotion.freight?.name} is being promoted`}
              >
                <FontAwesomeIcon icon={faGear} spin={true} />
              </Tooltip>
            ) : stage.status?.phase === 'Verifying' ? (
              <Tooltip title='Verifying Current Freight'>
                <FontAwesomeIcon icon={faCircleNotch} spin={true} />
              </Tooltip>
            ) : (
              stage.status?.health && (
                <HealthStatusIcon
                  health={stage.status?.health}
                  style={{ fontSize: '14px' }}
                  hideColor={true}
                />
              )
            )}
          </div>
        </h3>
        <div
          className={styles.body}
          style={currentFreight && currentFreight?.length > 1 ? { paddingTop: '15px' } : undefined}
        >
          {action === FreightTimelineAction.ManualApproval ||
          action === FreightTimelineAction.PromoteFreight ? (
            <div className='h-full flex items-center justify-center font-bold cursor-pointer text-blue-500 hover:text-blue-400'>
              <Button
                icon={
                  <FontAwesomeIcon
                    icon={
                      action === FreightTimelineAction.ManualApproval ? faCircleCheck : faArrowRight
                    }
                  />
                }
                disabled={stage?.spec?.promotionMechanisms === undefined}
                className='uppercase'
              >
                {action === FreightTimelineAction.ManualApproval ? 'Approve' : 'Promote'}
              </Button>
            </div>
          ) : (
            <div className='text-sm h-full flex flex-col items-center justify-center'>
              <FreightIndicators
                freight={currentFreight}
                selectedFreight={visibleFreight}
                onClick={(idx) => setVisibleFreight(idx)}
              />
              <FreightLabel freight={currentFreight[visibleFreight]} />
            </div>
          )}
        </div>
        <StageNodeFooter lastPromotion={stage?.status?.lastPromotion?.finishedAt?.toDate()} />
      </div>
      {action !== FreightTimelineAction.ManualApproval &&
        action !== FreightTimelineAction.PromoteFreight && (
          <>
            {stage.spec?.promotionMechanisms && (
              <Nodule
                begin={true}
                nodeHeight={height}
                onClick={() => onPromoteClick(FreightTimelineAction.Promote)}
                selected={action === FreightTimelineAction.Promote}
              />
            )}
            {!hasNoSubscribers && (
              <Nodule
                nodeHeight={height}
                onClick={() => onPromoteClick(FreightTimelineAction.PromoteSubscribers)}
                selected={action === FreightTimelineAction.PromoteSubscribers}
              />
            )}
          </>
        )}
    </>
  );
};

export const Nodule = (props: {
  begin?: boolean;
  nodeHeight: number;
  onClick?: () => void;
  selected?: boolean;
  icon?: IconDefinition;
}) => {
  const noduleHeight = 30;
  const top = props.nodeHeight / 2 - noduleHeight / 2;
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
