import {
  IconDefinition,
  faBullseye,
  faCircleCheck,
  faCircleNotch,
  faGear,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';
import { formatDistance } from 'date-fns';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { FreightLabel } from '@ui/features/common/freight-label';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { FreightTimelineAction } from '../types';

import * as styles from './stage-node.module.less';

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
  approving,
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
  currentFreight: Freight;
  onClick?: () => void;
  approving?: boolean;
  onHover: (hovering: boolean) => void;
  highlighted?: boolean;
}) => {
  const navigate = useNavigate();
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
        <div className={styles.body}>
          {approving ? (
            <div className='h-full flex items-center justify-center font-bold cursor-pointer text-blue-500 hover:text-blue-400'>
              <Button icon={<FontAwesomeIcon icon={faCircleCheck} />}>APPROVE</Button>
            </div>
          ) : (
            <div className='text-sm h-full flex flex-col items-center justify-center -mt-1'>
              <div className={styles.freightLabel}>Current Freight</div>
              <FreightLabel freight={currentFreight} showContents={true} />
              {stage?.status?.lastPromotion?.finishedAt && (
                <>
                  <div
                    className='uppercase font-medium mt-1 text-gray-400'
                    style={{ fontSize: '9px' }}
                  >
                    Last Promoted
                  </div>
                  <div className='text-xs text-gray-600 font-mono font-semibold'>
                    {formatDistance(
                      stage?.status?.lastPromotion?.finishedAt?.toDate(),
                      new Date(),
                      { addSuffix: true }
                    )}
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </div>
      {!approving && (
        <>
          <Nodule
            begin={true}
            nodeHeight={height}
            onClick={() => onPromoteClick(FreightTimelineAction.Promote)}
            selected={action === FreightTimelineAction.Promote}
          />
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
