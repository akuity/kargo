import { Tooltip } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { ColorMap } from '@ui/features/stage/utils';
import { Health, Stage } from '@ui/gen/api/v2/models';

// StagePill is a Stage as the pipeline paints it -- a coloured chip with the
// Stage's name -- linking to the Stage's drawer, which opens on its Fleet tab
// for a target-aware Stage. When the Target's health under this Stage is
// known, it leads the chip.
export const StagePill = ({
  projectName,
  stage,
  health,
  stageColorMap
}: {
  projectName: string;
  stage: Stage;
  health?: Health;
  stageColorMap: ColorMap;
}) => {
  const stageName = stage.metadata?.name || '';
  return (
    <Tooltip title={`Open ${stageName} on its Fleet tab`} placement='bottom'>
      <Link
        to={generatePath(paths.stage, { name: projectName, stageName })}
        className='inline-flex items-center gap-2 text-white rounded py-1 px-2 font-semibold text-xs bg-gray-600 hover:text-white hover:opacity-90'
        style={{ backgroundColor: stageColorMap[stageName] }}
      >
        {health?.status && <HealthStatusIcon health={health} hideColor noTooltip />}
        {stageName}
      </Link>
    </Tooltip>
  );
};
