import { Flex } from 'antd';
import { ColumnType } from 'antd/es/table';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { getCurrentFreight } from '@ui/features/common/utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const phaseColumn = (): ColumnType<Stage> => ({
  title: 'Phase',
  width: '10%',
  render: (_, stage) => {
    const stagePhase = getStagePhase(stage);

    if (getCurrentFreight(stage).length > 0) {
      const Comp = (
        <Flex align='center' gap={4}>
          {stagePhase}{' '}
          <StageConditionIcon
            conditions={stage?.status?.conditions || []}
            noTooltip
            className='text-[10px]'
          />
        </Flex>
      );

      if (stagePhase === 'Promoting') {
        return (
          <Link
            to={generatePath(paths.promotion, {
              name: stage?.metadata?.namespace,
              promotionId: stage?.status?.currentPromotion?.name
            })}
          >
            {Comp}
          </Link>
        );
      }

      return Comp;
    }

    return '-';
  }
});
