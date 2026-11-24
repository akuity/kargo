import { faFilter } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Badge, Button, Popover } from 'antd';
import classNames from 'classnames';

import {
  FreightTimelineControllerContextType,
  useFreightTimelineControllerContext
} from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v2/models';

import { FreightTimelineFilters } from './freight-timeline-filters';

type Props = {
  className?: string;
  freights: Freight[];
  // names of the project's Warehouses
  warehouses: string[];
};

// These narrow the freight timeline only, not the graph beside it.
const label = 'Freight timeline filters';

// The Warehouse the freight timeline is narrowed to does not count: the
// timeline already says so, with the Warehouse's name and an arrow back.
const isFilterActive = (
  preferredFilter: FreightTimelineControllerContextType['preferredFilter']
): boolean =>
  (preferredFilter?.sources?.length ?? 0) > 0 ||
  preferredFilter?.timerange !== 'all-time' ||
  preferredFilter?.hideUnusedFreights === true;

export const FreightTimelineFilterButton = (props: Props) => {
  const ctx = useFreightTimelineControllerContext();

  if (!ctx) {
    throw new Error('missing context freightTimelineControllerContext');
  }

  const active = isFilterActive(ctx.preferredFilter);

  return (
    <Popover
      trigger='click'
      placement='bottomRight'
      title={label}
      content={
        <FreightTimelineFilters
          preferredFilter={ctx.preferredFilter}
          onPreferredFilterChange={ctx.setPreferredFilter}
          freights={props.freights}
          warehouses={props.warehouses}
        />
      }
    >
      <Badge dot={active} offset={[-4, 4]}>
        <Button
          aria-label={label}
          title={label}
          className={classNames(props.className)}
          icon={<FontAwesomeIcon icon={faFilter} />}
        />
      </Badge>
    </Popover>
  );
};
