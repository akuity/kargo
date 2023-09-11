import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

import * as styles from './stage-node.module.less';

export const StageNode = ({
  stage,
  color,
  height,
  onPromoteSubscribersClick
}: {
  stage: Stage;
  color: string;
  height: number;
  onPromoteSubscribersClick: () => void;
}) => (
  <div className={styles.node} style={{ backgroundColor: color, position: 'relative' }}>
    <h3 className='flex items-center text-white'>
      <div>{stage.metadata?.name}</div>
      {stage.status?.health && (
        <HealthStatusIcon
          health={stage.status?.health}
          style={{ marginLeft: 'auto', fontSize: '14px' }}
          hideColor={true}
        />
      )}
    </h3>
    <div className={styles.body}>
      <h3>Current Freight</h3>
      <p className='font-mono text-sm font-semibold'>
        {stage.status?.currentFreight?.id?.slice(0, 7) || 'N/A'}{' '}
      </p>
    </div>
    <Nodule nodeHeight={height} onClick={onPromoteSubscribersClick} />
  </div>
);

const Nodule = (props: { begin?: boolean; nodeHeight: number; onClick?: () => void }) => {
  const noduleHeight = 16;
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
        width: noduleHeight
      }}
      className={`z-10 bg-gray-400 hover:bg-blue-400 absolute ${
        props.begin ? '-left-2' : '-right-2'
      } rounded-md`}
    />
  );
};
