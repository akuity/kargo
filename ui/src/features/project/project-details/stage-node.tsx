import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

import * as styles from './stage-node.module.less';

export const StageNode = ({ stage, color }: { stage: Stage; color: string }) => (
  <div className={styles.node} style={{ backgroundColor: color }}>
    <h3>
      {stage.metadata?.name}
      {stage.status?.health && (
        <HealthStatusIcon
          health={stage.status?.health}
          style={{ position: 'absolute', right: '1em' }}
        />
      )}
    </h3>
    <div className={styles.body}>
      <h3>Current Freight</h3>
      <p className='font-mono text-sm font-semibold'>
        {stage.status?.currentFreight?.id?.slice(0, 7) || 'N/A'}{' '}
      </p>
    </div>
  </div>
);
