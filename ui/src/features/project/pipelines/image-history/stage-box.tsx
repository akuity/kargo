import { Button, Tooltip } from 'antd';
import { memo } from 'react';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';

const getTooltipTitle = (
  showHistory: boolean,
  isHighlighted: boolean,
  stageName: string,
  orders: number[]
): string => {
  if (!showHistory) {
    return `Promoted to stage: '${stageName}'`;
  }

  if (orders.length === 1) {
    if (isHighlighted) {
      return `Most recent promotion: Currently in stage '${stageName}'`;
    }
    return `Stage: '${stageName}' (Promotion order: ${orders[0]})`;
  }

  if (isHighlighted) {
    return `Most recent promotion: Currently in stage '${stageName}'. All promotions: ${orders.join(', ')}`;
  }

  return `Stage: '${stageName}' (${orders.length} promotions: ${orders.join(', ')})`;
};

export const StageBox = memo(
  ({
    stageName,
    orders,
    showHistory,
    stageColorMap,
    project
  }: {
    stageName: string;
    orders: number[];
    showHistory: boolean;
    stageColorMap: Record<string, string>;
    project: string;
  }) => {
    const navigate = useNavigate();
    const isHighlighted = showHistory && orders.includes(1);
    const baseColor = stageColorMap[stageName] || '#6b7280';

    const tooltipTitle = getTooltipTitle(showHistory, isHighlighted, stageName, orders);

    const handleClick = () => {
      navigate(generatePath(paths.stage, { name: project, stageName }));
    };

    const style = {
      backgroundColor: baseColor,
      opacity: showHistory && !isHighlighted ? 0.6 : 1,
      border: isHighlighted ? '2px solid rgba(0,0,0,0.4)' : '2px solid transparent',
      boxShadow: isHighlighted ? '0 1px 3px rgba(0, 0, 0, 0.2)' : 'none'
    };

    return (
      <Tooltip title={tooltipTitle}>
        <Button
          onClick={handleClick}
          className='h-6 w-full rounded flex items-center justify-center cursor-pointer transition-all duration-300 p-0'
          style={style}
        >
          {showHistory && (
            <div className='flex items-center gap-0.5'>
              {orders.length === 1 ? (
                <span className='text-white font-bold text-[10px] select-none'>{orders[0]}</span>
              ) : (
                <div className='flex items-center gap-0.5'>
                  {orders.slice(0, 3).map((order, index) => (
                    <span
                      key={`${index}-${order}`}
                      className='text-white font-bold text-[10px] select-none'
                    >
                      {order}
                      {index < Math.min(3, orders.length) - 1 ? ',' : ''}
                    </span>
                  ))}
                  {orders.length > 3 && (
                    <span className='text-white font-bold text-[8px] select-none'>...</span>
                  )}
                </div>
              )}
            </div>
          )}
        </Button>
      </Tooltip>
    );
  }
);
