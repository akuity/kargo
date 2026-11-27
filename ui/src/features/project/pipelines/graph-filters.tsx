import { faCircleNodes, faList, faSearch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Card, Input, Segmented, Select, Space } from 'antd';
import classNames from 'classnames';

import { WarehouseExpanded } from '@ui/extend/types';
import { Freight, Stage } from '@ui/gen/api/v2/models';

import { useFreightTimelineControllerContext } from './context/freight-timeline-controller-context';
import { DisplaySettingsButton } from './display-settings-button';
import { floatingSurfaceBorderColor, floatingSurfaceClassName } from './floating-surface';
import { FreightTimelineFilterButton } from './freight/freight-timeline-filter-button';

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

  return (
    <Card
      size='small'
      // floats over the graph like the freight timeline, on the same surface
      className={classNames(floatingSurfaceClassName, props.className)}
      style={{ borderColor: floatingSurfaceBorderColor }}
      styles={{ body: { padding: 8 } }}
    >
      <Space size={8}>
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
              warehouses,
              // the freight timeline's own Warehouse may now be filtered out
              freightTimelineWarehouse: ''
            })
          }
        />

        <Input
          allowClear
          className='w-[180px]'
          placeholder='Search stages...'
          prefix={
            <FontAwesomeIcon
              icon={faSearch}
              className='text-xs text-gray-400 dark:text-neutral-500'
            />
          }
          value={filterContext?.stageSearch || ''}
          onChange={(e) => filterContext?.setStageSearch(e.target.value)}
        />

        <Segmented
          // The stock track is about the color of this translucent toolbar
          // in both themes; darken it so the switch reads as one control.
          className='!bg-black/[0.08] dark:!bg-black/30'
          value={props.pipelineView}
          options={[
            { value: 'graph', icon: <FontAwesomeIcon icon={faCircleNodes} /> },
            { value: 'list', icon: <FontAwesomeIcon icon={faList} /> }
          ]}
          onChange={props.setPipelineView}
        />

        <FreightTimelineFilterButton
          freights={props.freights}
          warehouses={props.warehouses.map((warehouse) => warehouse?.metadata?.name || '')}
        />

        <DisplaySettingsButton
          stages={props.stages}
          warehouses={props.warehouses}
          pipelineView={props.pipelineView}
        />
      </Space>
    </Card>
  );
};
