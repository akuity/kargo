import { faObjectGroup } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Select, Typography } from 'antd';

import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { useFreightTimelineControllerContext } from './context/freight-timeline-controller-context';
import { groupNodes } from './group-nodes';

type GraphFiltersProps = {
  warehouses: Warehouse[];
  stages: Stage[];
};

export const GraphFilters = (props: GraphFiltersProps) => {
  const filterContext = useFreightTimelineControllerContext();

  return (
    <Card size='small'>
      <Typography.Text className='text-xs' type='secondary'>
        Warehouses:{' '}
      </Typography.Text>
      <Select
        size='small'
        mode='multiple'
        className='ml-1 min-w-[300px]'
        maxTagCount={2}
        placeholder='All'
        value={filterContext?.preferredFilter?.warehouses || []}
        options={props.warehouses.map((warehouse) => ({
          label: warehouse?.metadata?.name,
          value: warehouse?.metadata?.name
        }))}
        onChange={(warehouses) =>
          filterContext?.setPreferredFilter({
            ...filterContext?.preferredFilter,
            warehouses
          })
        }
      />

      <Button
        className='ml-3'
        title='Group stages'
        icon={<FontAwesomeIcon icon={faObjectGroup} />}
        onClick={() => {
          filterContext?.setPreferredFilter({
            ...filterContext?.preferredFilter,
            stackedNodesParents: groupNodes(props.stages, props.warehouses)
          });
        }}
      />
    </Card>
  );
};
