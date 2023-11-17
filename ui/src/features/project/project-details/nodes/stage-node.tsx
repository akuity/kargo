import { faBullseye, faGear, faTruckArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionType } from '@ui/features/freightline/freightline';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

import * as styles from './stage-node.module.less';

export const StageNode = ({
  stage,
  color,
  height,
  faded,
  onPromoteClick,
  projectName,
  hasNoSubscribers,
  promoting
}: {
  stage: Stage;
  color: string;
  height: number;
  faded: boolean;
  projectName?: string;
  hasNoSubscribers?: boolean;
  promoting?: PromotionType;
  onPromoteClick: (type: PromotionType) => void;
}) => {
  const navigate = useNavigate();
  return (
    <div
      className={styles.node}
      style={{ backgroundColor: color, position: 'relative', cursor: 'pointer' }}
      onClick={() =>
        navigate(generatePath(paths.stage, { name: projectName, stageName: stage.metadata?.name }))
      }
    >
      <div
        className={`${styles.node} ${faded ? styles.faded : ''}`}
        style={{
          backgroundColor: color,
          position: 'relative'
        }}
      >
        <h3 className='flex items-center text-white justify-between'>
          <div className='text-ellipsis whitespace-nowrap overflow-hidden h-8'>
            {stage.metadata?.name}
          </div>
          {stage.status?.currentPromotion ? (
            <Tooltip
              title={`Freight ${stage.status?.currentPromotion.freight?.id} is being promoted`}
            >
              <FontAwesomeIcon icon={faGear} spin={true} />
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
        </h3>
        <div className={styles.body}>
          <h3>Current Freight</h3>
          <p className='font-mono text-sm font-semibold'>
            {stage.status?.currentFreight?.id?.slice(0, 7) || 'N/A'}{' '}
          </p>
        </div>
        <>
          <Nodule
            begin={true}
            nodeHeight={height}
            onClick={() => onPromoteClick('default')}
            selected={promoting === 'default'}
          />
          {!hasNoSubscribers && (
            <Nodule
              nodeHeight={height}
              onClick={() => onPromoteClick('subscribers')}
              selected={promoting === 'subscribers'}
            />
          )}
        </>
      </div>
    </div>
  );
};

const Nodule = (props: {
  begin?: boolean;
  nodeHeight: number;
  onClick?: () => void;
  selected?: boolean;
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
      className={`z-10 flex items-center justify-center hover:text-white border border-sky-300 border-solid hover:bg-blue-400 absolute rounded-lg ${
        props.selected ? 'text-white bg-blue-400' : 'bg-white text-blue-500'
      }`}
    >
      <FontAwesomeIcon icon={props.begin ? faBullseye : faTruckArrowRight} />
    </div>
  );
};
