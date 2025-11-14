import { Tooltip } from 'antd';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { StagePopover } from '../project/list/project-item/stage-popover';
import { ColorMap } from '../stage/utils';

import { HealthStatusIcon } from './health-status/health-status-icon';
import { PromotionStatusIcon } from './promotion-status/promotion-status-icon';

export const StageTag = ({
  stage,
  projectName,
  stageColorMap
}: {
  stage: Stage;
  projectName: string;
  stageColorMap: ColorMap;
}) => {
  return (
    <Tooltip
      key={stage.metadata?.name}
      placement='bottom'
      title={
        stage?.status?.lastPromotion?.name && <StagePopover project={projectName} stage={stage} />
      }
    >
      <div
        className='flex items-center mb-2 text-white rounded py-1 px-2 font-semibold bg-gray-600'
        style={{ backgroundColor: stageColorMap[stage.metadata?.name || ''] }}
      >
        {stage.status?.health && (
          <div className='mr-2'>
            <HealthStatusIcon health={stage.status?.health} hideColor={true} />
          </div>
        )}
        {!stage?.status?.currentPromotion && stage.status?.lastPromotion && (
          <div className='mr-2'>
            <PromotionStatusIcon
              placement='top'
              status={stage.status?.lastPromotion?.status}
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
