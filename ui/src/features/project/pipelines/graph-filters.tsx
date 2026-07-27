import { faCircleNodes, faList, faObjectGroup, faSearch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Input, Segmented, Select, Space } from 'antd';
import { useMemo } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { Freight, Stage } from '@ui/gen/api/v2/models';

import { useFreightTimelineControllerContext } from './context/freight-timeline-controller-context';
import { FreightTimelineFilterButton } from './freight/freight-timeline-filter-button';
import { groupNodes } from './group-nodes';

type GraphFiltersProps = {
  warehouses: WarehouseExpanded[];
  stages: Stage[];
  freights: Freight[];
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
      <Space size={12}>
        <Select
          mode='multiple'
          className='min-w-[240px]'
          maxTagCount={2}
          placeholder='All Warehouses'
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

        <Input
          allowClear
          className='w-[180px]'
          placeholder='Search stages...'
          prefix={<FontAwesomeIcon icon={faSearch} className='text-xs text-gray-400' />}
          value={filterContext?.stageSearch || ''}
          onChange={(e) => filterContext?.setStageSearch(e.target.value)}
        />

        <FreightTimelineFilterButton freights={props.freights} />

        <Button
          title='Group stages'
          icon={<FontAwesomeIcon icon={faObjectGroup} />}
          onClick={() => {
            filterContext?.setPreferredFilter({
              ...filterContext?.preferredFilter,
              stackedNodesParents
            });
          }}
          disabled={stackedNodesParents.length < 1 || props.pipelineView !== 'graph'}
        />

        <Segmented
          value={props.pipelineView}
          options={[
            { value: 'graph', icon: <FontAwesomeIcon icon={faCircleNodes} /> },
            { value: 'list', icon: <FontAwesomeIcon icon={faList} /> }
          ]}
          onChange={props.setPipelineView}
        />
      </Space>
    </Card>
  );
};
