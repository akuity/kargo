import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Checkbox, Select, SelectProps } from 'antd';
import classNames from 'classnames';
import { useContext, useMemo } from 'react';

import { ColorContext } from '@ui/context/colors';
import { FreightTimelineControllerContextType } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v2/models';

import { WarehouseIcon } from '../warehouse-icon';

import { humanComprehendableArtifact } from './artifact-parts-utils';
import { timerangeOrderedOptions, timerangeToLabel } from './filter-timerange-utils';
import { getFocusableWarehouses } from './freight-warehouse-group-utils';
import { catalogueFreights } from './source-catalogue-utils';

type FreightTimelineFiltersProps = {
  className?: string;
  preferredFilter: FreightTimelineControllerContextType['preferredFilter'];
  onPreferredFilterChange(next: FreightTimelineControllerContextType['preferredFilter']): void;
  freights: Freight[];
  // names of the project's Warehouses, to narrow the freight timeline to one
  warehouses: string[];
};

export const FreightTimelineFilters = (props: FreightTimelineFiltersProps) => {
  const { warehouseColorMap } = useContext(ColorContext);

  const focusableWarehouses = getFocusableWarehouses(
    props.warehouses,
    props.preferredFilter?.warehouses || []
  );
  const focusedWarehouse = props.preferredFilter?.freightTimelineWarehouse;

  const sourcesDropdownOptions: SelectProps['options'] = useMemo(() => {
    const freightSourcesCatalogue = catalogueFreights(props.freights);

    const opts: SelectProps['options'] = [];

    for (const [sourceType, repoURLs] of Object.entries(freightSourcesCatalogue)) {
      let icon: IconDefinition = faGitAlt;

      if (sourceType === 'images') {
        icon = faDocker;
      } else if (sourceType === 'charts') {
        icon = faAnchor;
      }

      for (const repoURL of repoURLs) {
        opts.push({
          value: repoURL,
          label: (
            <div className='w-fit'>
              <FontAwesomeIcon icon={icon} className='mr-2' />
              <span className='text-xs'>{repoURL}</span>
            </div>
          )
        });
      }
    }

    return opts;
  }, [props.freights]);

  return (
    <div className={classNames('min-w-[300px]', props.className)}>
      <div className='text-xs flex items-center gap-3'>
        <label>Source: </label>
        <Select
          mode='multiple'
          className='min-w-[200px] ml-auto'
          styles={{ popup: { root: { width: '500px' } } }}
          size='small'
          value={props.preferredFilter?.sources}
          onChange={(sources) =>
            props.onPreferredFilterChange({ ...props.preferredFilter, sources })
          }
          labelRender={(props) => humanComprehendableArtifact({ repoURL: props.value.toString() })}
          placeholder='All'
          options={sourcesDropdownOptions}
          maxTagCount={1}
        />
      </div>
      <div className='text-xs flex items-center gap-3 mt-2'>
        <label>Timerange: </label>
        <Select
          className='min-w-[200px] ml-auto'
          size='small'
          value={props.preferredFilter?.timerange}
          options={timerangeOrderedOptions.map((opt) => ({
            value: opt,
            label: <>{timerangeToLabel(opt)}</>
          }))}
          maxTagCount={1}
          onChange={(timerange) =>
            props.onPreferredFilterChange({ ...props.preferredFilter, timerange: timerange })
          }
        />
      </div>

      <div className='text-xs flex items-center gap-3 mt-2'>
        <label>Warehouse: </label>
        <Select
          allowClear
          className='min-w-[200px] ml-auto'
          size='small'
          placeholder='All'
          // with one Warehouse, every piece of freight is already its own;
          // still clearable while set, e.g. to a Warehouse since deleted
          disabled={focusableWarehouses.length < 2 && !focusedWarehouse}
          value={focusedWarehouse || undefined}
          options={focusableWarehouses.map((warehouse) => ({
            value: warehouse,
            label: (
              <span className='flex items-center gap-1.5'>
                <WarehouseIcon color={warehouseColorMap?.[warehouse]} />
                {warehouse}
              </span>
            )
          }))}
          onChange={(warehouse) =>
            props.onPreferredFilterChange({
              ...props.preferredFilter,
              freightTimelineWarehouse: warehouse || ''
            })
          }
        />
      </div>

      <div className='mt-3'>
        <Checkbox
          className='text-xs'
          checked={props.preferredFilter?.hideUnusedFreights}
          onChange={(e) =>
            props.onPreferredFilterChange({
              ...props.preferredFilter,
              hideUnusedFreights: e.target.checked
            })
          }
        >
          Hide unused freights
        </Checkbox>
      </div>
    </div>
  );
};
