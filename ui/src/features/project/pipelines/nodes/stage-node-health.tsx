import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { healthTone } from '@ui/features/common/health-status/utils';
import { useToneColor } from '@ui/features/common/status-tone';
import { Stage } from '@ui/gen/api/v2/models';

import './stage-node.less';

import { getStageHealth } from './stage-meta-utils';

// StageNodeHealth states the health a Stage reports, as one column of the
// node's status band. The icon takes the column's color rather than its own,
// so the two never disagree by a shade.
export const StageNodeHealth = (props: { stage: Stage }) => {
  const health = getStageHealth(props.stage);

  // A Stage reporting nothing resolves to the muted tone, which is the tone
  // the dash below wants anyway.
  const color = useToneColor(healthTone(health?.status));

  // The band keeps two columns whether or not the Stage reports health, so an
  // absent status is stated as such rather than leaving a hole.
  if (!health?.status) {
    return (
      <span
        className='stage-node__status-value'
        style={{ color, fontWeight: 400 }}
        title='No health reported'
      >
        &mdash;
      </span>
    );
  }

  return (
    <span className='stage-node__status-value' style={{ color }}>
      {/* the tooltip would otherwise repeat the word next to it; it earns its
          place only when the Stage reported issues to go with the status */}
      <HealthStatusIcon
        health={health}
        className='text-[9px]'
        hideColor
        noTooltip={!health.issues?.length}
      />
      {health.status}
    </span>
  );
};
