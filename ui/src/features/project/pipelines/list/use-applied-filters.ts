import { useMemo } from 'react';

import { useFilterContext } from './context/filter-context';

type AppliedFilter = {
  key: string;
  value: string;
  onClear(): void;
};

export const useAppliedFilters = () => {
  const filterContext = useFilterContext();

  return useMemo(() => {
    const filters = filterContext?.filters;

    const appliedFilters: AppliedFilter[] = [];

    if (filters?.stage) {
      appliedFilters.push({
        key: 'stage',
        value: filters.stage,
        onClear: () => filterContext?.onFilter({ ...filterContext?.filters, stage: '' })
      });
    }

    return appliedFilters;
  }, [filterContext?.filters, filterContext?.onFilter]);
};
