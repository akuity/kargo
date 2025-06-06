import { Flex } from 'antd';
import { ColumnType } from 'antd/es/table';

import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const healthColumn = (): ColumnType<Stage> => ({
  title: 'Health',
  width: '10%',
  render: (_, stage) => {
    const stageHealth = stage?.status?.health;

    if (stageHealth?.status) {
      return (
        <Flex gap={4} align='center'>
          {stageHealth?.status}
          <HealthStatusIcon noTooltip health={stageHealth} />
        </Flex>
      );
    }

    return '-';
  }
});
