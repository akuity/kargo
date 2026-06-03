import { faFilter } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Badge, Button, Popover } from 'antd';
import classNames from 'classnames';

import {
  FreightTimelineControllerContextType,
  useFreightTimelineControllerContext
} from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { FreightTimelineFilters } from './freight-timeline-filters';

type Props = {
  className?: string;
  freights: Freight[];
};

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
      content={
        <FreightTimelineFilters
          preferredFilter={ctx.preferredFilter}
          onPreferredFilterChange={ctx.setPreferredFilter}
          freights={props.freights}
        />
      }
    >
      <Badge dot={active} offset={[-4, 4]}>
        <Button
          className={classNames(props.className)}
          icon={<FontAwesomeIcon icon={faFilter} />}
        />
      </Badge>
    </Popover>
  );
};
