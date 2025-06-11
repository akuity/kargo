import { Button, Checkbox, Flex } from 'antd';
import { ColumnType } from 'antd/es/table';

import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import {
  Filter,
  useFilterContext
} from '@ui/features/project/pipelines/list/context/filter-context';
import { getStageHealth } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const healthColumn = (filter: Filter): ColumnType<Stage> => ({
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
  },
  filterDropdown: (props) => {
    const filters = useFilterContext();

    const onApply = () =>
      filters?.onFilter({
        ...filters.filters,
        health: (props.selectedKeys as string[]) || []
      });

    return (
      <div style={{ padding: 8 }}>
        <Flex vertical gap={8}>
          {['Healthy', 'Progressing', 'Unhealthy', 'Unknown'].map((healthStatus) => (
            <Checkbox
              key={healthStatus}
              value={healthStatus}
              checked={props.selectedKeys.includes(healthStatus)}
              onChange={(e) => {
                const selected = props.selectedKeys.includes(healthStatus);

                if (e.target.checked && !selected) {
                  props.setSelectedKeys([...props.selectedKeys, healthStatus]);
                  return;
                }

                if (!e.target.checked && selected) {
                  props.setSelectedKeys(props.selectedKeys.filter((k) => k !== healthStatus));
                }
              }}
            >
              {healthStatus}
            </Checkbox>
          ))}

          <Button type='primary' size='small' onClick={onApply}>
            Apply
          </Button>
        </Flex>
      </div>
    );
  },
  filteredValue: filter?.health || [],
  onFilter: (_, record) => {
    const health = filter?.health;

    const healthStatus = getStageHealth(record)?.status || '';

    return health?.includes(healthStatus);
  },
  filterMultiple: true
});
