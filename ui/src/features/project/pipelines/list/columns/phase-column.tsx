import { Button, Flex } from 'antd';
import Checkbox from 'antd/es/checkbox/Checkbox';
import { ColumnType } from 'antd/es/table';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { getCurrentFreight } from '@ui/features/common/utils';
import {
  Filter,
  useFilterContext
} from '@ui/features/project/pipelines/list/context/filter-context';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const phaseColumn = (filter: Filter): ColumnType<Stage> => ({
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
  },
  filterDropdown: (props) => {
    const filters = useFilterContext();

    const onApply = () =>
      filters?.onFilter({
        ...filters.filters,
        phase: (props.selectedKeys as string[]) || []
      });

    return (
      <div style={{ padding: 8 }}>
        <Flex vertical gap={8}>
          {['Promoting', 'Verifying', 'Reconciling', 'Failed', 'Ready', 'Steady'].map((phase) => (
            <Checkbox
              key={phase}
              value={phase}
              checked={props.selectedKeys.includes(phase)}
              onChange={(e) => {
                const selected = props.selectedKeys.includes(phase);

                if (e.target.checked && !selected) {
                  props.setSelectedKeys([...props.selectedKeys, phase]);
                  return;
                }

                if (!e.target.checked && selected) {
                  props.setSelectedKeys(props.selectedKeys.filter((k) => k !== phase));
                }
              }}
            >
              {phase}
            </Checkbox>
          ))}
          <Button type='primary' size='small' onClick={onApply}>
            Apply
          </Button>
        </Flex>
      </div>
    );
  },
  filteredValue: filter?.phase || [],
  onFilter: (_, record) => {
    const phase = filter?.phase;

    const stagePhase = getStagePhase(record);

    return phase?.includes(stagePhase);
  },
  filterMultiple: true
});
