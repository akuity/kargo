import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

import * as styles from './stage-item.module.less';

export const StageItem = (props: { stage: Stage; onClick: () => void }) => {
  const { stage } = props;

  return (
    <div key={stage.metadata?.name} onClick={props.onClick} className={styles.item}>
      <HealthStatusIcon
        health={stage.status?.currentFreight?.health}
        style={{ marginRight: '12px' }}
      />
      {stage.metadata?.name}
    </div>
  );
};
