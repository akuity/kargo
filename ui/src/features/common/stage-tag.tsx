import { Tooltip } from 'antd';

import {
  getCurrentPromotionRef,
  getLastPromotionRef
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Health, Stage } from '@ui/gen/api/v2/models';

import { StagePopover } from '../project/list/project-item/stage-popover';
import { ColorMap } from '../stage/utils';

import { HealthStatusIcon } from './health-status/health-status-icon';
import { PromotionStatusIcon } from './promotion-status/promotion-status-icon';

export const StageTag = ({
  stage,
  projectName,
  stageColorMap,
  health = stage.status?.health
}: {
  stage: Stage;
  projectName: string;
  stageColorMap: ColorMap;
  // health overrides the Stage's own health for surfaces that show one
  // Target's standing under the Stage rather than the Stage as a whole.
  health?: Health;
}) => {
  const currentPromotion = getCurrentPromotionRef(stage);
  const lastPromotion = getLastPromotionRef(stage);

  return (
    <Tooltip
      key={stage.metadata?.name}
      placement='bottom'
      title={lastPromotion?.name && <StagePopover project={projectName} stage={stage} />}
    >
      <div
        className='flex items-center mb-2 text-white rounded py-1 px-2 font-semibold bg-gray-600'
        style={{ backgroundColor: stageColorMap[stage.metadata?.name || ''] }}
      >
        {health && (
          <div className='mr-2'>
            <HealthStatusIcon health={health} hideColor={true} />
          </div>
        )}
        {!currentPromotion && lastPromotion && (
          <div className='mr-2'>
            <PromotionStatusIcon
              placement='top'
              status={{ phase: lastPromotion.phase, message: lastPromotion.message }}
              subject={
                lastPromotion.kind === 'PromotionRequest' ? 'Promotion Request' : 'Promotion'
              }
              color='white'
              size='1x'
            />
          </div>
        )}
        {stage.metadata?.name}
      </div>
    </Tooltip>
  );
};
