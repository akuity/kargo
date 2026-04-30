import { faCircleNodes, faList, faObjectGroup } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Segmented, Select, Typography } from 'antd';
import { useMemo } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { useFreightTimelineControllerContext } from './context/freight-timeline-controller-context';
import { groupNodes } from './group-nodes';

type GraphFiltersProps = {
  warehouses: WarehouseExpanded[];
  stages: Stage[];
  pipelineView: 'graph' | 'list';
  setPipelineView: (view: 'graph' | 'list') => void;
  className?: string;
};

export const GraphFilters = (props: GraphFiltersProps) => {
  const filterContext = useFreightTimelineControllerContext();

  const stackedNodesParents = useMemo(
    () => groupNodes(props.stages, props.warehouses).filter(Boolean),
    [props.stages, props.warehouses]
  );

  return (
    <Card size='small' className={props.className}>
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

      {props.pipelineView === 'graph' && (
        <Button
          className='ml-3'
          title='Group stages'
          icon={<FontAwesomeIcon icon={faObjectGroup} />}
          onClick={() => {
            filterContext?.setPreferredFilter({
              ...filterContext?.preferredFilter,
              stackedNodesParents
            });
          }}
          disabled={stackedNodesParents.length < 1}
        />
      )}

      <Segmented
        className='ml-3'
        value={props.pipelineView}
        options={[
          { value: 'graph', icon: <FontAwesomeIcon icon={faCircleNodes} /> },
          { value: 'list', icon: <FontAwesomeIcon icon={faList} /> }
        ]}
        onChange={props.setPipelineView}
      />
    </Card>
  );
};
