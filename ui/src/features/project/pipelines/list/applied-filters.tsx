import { Tag } from 'antd';
import classNames from 'classnames';

import { useAppliedFilters } from './use-applied-filters';

type AppliedFiltersProps = {
  className?: string;
};

export const AppliedFilters = (props: AppliedFiltersProps) => {
  const appliedFilters = useAppliedFilters();

  if (!appliedFilters.length) {
    return null;
  }

  return (
    <div className={classNames(props.className, 'space-y-2')}>
      <span className='text-xs mr-2 font-bold'>Applied filters: </span>
      {appliedFilters.map((appliedFilter, idx) => (
        <Tag key={idx} onClose={appliedFilter.onClear} closeIcon>
          {appliedFilter.key}: {appliedFilter.value}
        </Tag>
      ))}
    </div>
  );
};
