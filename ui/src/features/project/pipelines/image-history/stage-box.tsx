import { Button, Tooltip } from 'antd';
import { memo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';

const getTooltipTitle = (
  showHistory: boolean,
  isHighlighted: boolean,
  stageName: string,
  order: number
): string => {
  if (!showHistory) {
    return `Promoted to stage: '${stageName}'`;
  }

  if (isHighlighted) {
    return `Most recent promotion: Currently in stage '${stageName}'`;
  }

  return `Stage: '${stageName}' (Promotion order: ${order})`;
};

export const StageBox = memo(
  ({
    stageName,
    order,
    showHistory,
    stageColorMap,
    project
  }: {
    stageName: string;
    order: number;
    showHistory: boolean;
    stageColorMap: Record<string, string>;
    project: string;
  }) => {
    const navigate = useNavigate();
    const isHighlighted = showHistory && order === 0;
    const baseColor = stageColorMap[stageName] || '#6b7280';

    const tooltipTitle = getTooltipTitle(showHistory, isHighlighted, stageName, order);

    const handleClick = () => {
      navigate(generatePath(paths.stage, { name: project, stageName }));
    };

    const style = {
      backgroundColor: baseColor,
      opacity: showHistory && !isHighlighted ? 0.6 : 1,
      border: isHighlighted ? '2px solid rgba(255,255,255,0.5)' : '2px solid transparent',
      boxShadow: isHighlighted ? '0 1px 3px rgba(0, 0, 0, 0.2)' : 'none'
    };

    return (
      <Tooltip title={tooltipTitle}>
        <Button
          onClick={handleClick}
          className='h-6 w-full rounded flex items-center justify-center cursor-pointer transition-all duration-300 border-0 p-0'
          style={style}
        >
          {showHistory && (
            <div className='flex items-center gap-0.5'>
              <span className='text-white font-bold text-[10px] select-none'>{order}</span>
            </div>
          )}
        </Button>
      </Tooltip>
    );
  }
);
