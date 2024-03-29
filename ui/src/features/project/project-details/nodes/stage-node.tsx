import {
  IconDefinition,
  faBullseye,
  faCircleCheck,
  faCircleNotch,
  faGear,
  faTruckArrowRight
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { FreightLabel } from '@ui/features/common/freight-label';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { FreightlineAction } from '../types';

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
  action?: FreightlineAction;
  onPromoteClick: (type: FreightlineAction) => void;
  currentFreight: Freight;
  onClick?: () => void;
  approving?: boolean;
  onHover: (hovering: boolean) => void;
  highlighted?: boolean;
}) => {
  const navigate = useNavigate();
  return (
    <div
      className={styles.node}
      style={{ backgroundColor: color, position: 'relative', cursor: 'pointer' }}
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
      <div
        className={`${styles.node} ${faded ? styles.faded : ''} ${
          highlighted ? styles.highlighted : ''
        }`}
        style={{
          backgroundColor: color,
          position: 'relative'
        }}
      >
        <h3 className='flex items-center text-white justify-between'>
          <div className='truncate pb-1'>{stage.metadata?.name}</div>
          <div className='pb-1'>
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
              <FontAwesomeIcon icon={faCircleCheck} className='mr-2' />
              APPROVE
            </div>
          ) : (
            <>
              <h3>Current Freight</h3>
              <div className='font-mono text-sm font-semibold h-full flex items-center justify-center'>
                <FreightLabel freight={currentFreight} />
              </div>
            </>
          )}
        </div>
        {!approving && (
          <>
            <Nodule
              begin={true}
              nodeHeight={height}
              onClick={() => onPromoteClick('promote')}
              selected={action === 'promote'}
            />
            {!hasNoSubscribers && (
              <Nodule
                nodeHeight={height}
                onClick={() => onPromoteClick('promoteSubscribers')}
                selected={action === 'promoteSubscribers'}
              />
            )}
          </>
        )}
      </div>
    </div>
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
      className={`cursor-pointer select-none z-10 flex items-center justify-center hover:text-white border border-sky-300 border-solid hover:bg-blue-400 absolute rounded-lg ${
        props.selected ? 'text-white bg-blue-400' : 'bg-white text-blue-500'
      }`}
    >
      <FontAwesomeIcon
        icon={props.icon ? props.icon : props.begin ? faBullseye : faTruckArrowRight}
      />
    </div>
  );
};
