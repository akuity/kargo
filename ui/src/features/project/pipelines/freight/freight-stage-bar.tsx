import { Tooltip } from 'antd';

import { ColorMap } from '@ui/features/stage/utils';
import { Stage } from '@ui/gen/api/v2/models';

import { getStageBandColor } from './freight-stage-bar-utils';

type FreightStageBarProps = {
  stages: Stage[];
  stageColorMap: ColorMap;
  showColors: boolean;
};

// FreightStageBar marks which Stages hold a piece of freight: a 4px bar on the
// card's left edge, split vertically into one band per Stage in the color the
// Stage has in the graph. Hovering a band names its Stage. Renders nothing for
// freight no Stage holds.
export const FreightStageBar = ({ stages, stageColorMap, showColors }: FreightStageBarProps) => {
  if (stages.length === 0) {
    return null;
  }

  return (
    <div className='absolute left-0 top-0 bottom-0 w-1 flex flex-col gap-px'>
      {stages.map((stage) => {
        const name = stage?.metadata?.name || '';

        return (
          <Tooltip key={stage?.metadata?.uid || name} title={name} placement='left'>
            <div
              className='flex-1'
              style={{ background: getStageBandColor(name, stageColorMap, showColors) }}
            />
          </Tooltip>
        );
      })}
    </div>
  );
};
