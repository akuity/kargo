import { format } from 'date-fns';
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

    if (filters?.phase?.length) {
      for (const phase of filters.phase) {
        appliedFilters.push({
          key: 'phase',
          value: phase,
          onClear: () =>
            filterContext?.onFilter({
              ...filterContext?.filters,
              phase: filterContext?.filters?.phase?.filter((p) => p !== phase)
            })
        });
      }
    }

    if (filters?.health?.length) {
      for (const health of filters.health) {
        appliedFilters.push({
          key: 'health',
          value: health,
          onClear: () =>
            filterContext?.onFilter({
              ...filterContext?.filters,
              health: filterContext?.filters?.health?.filter((h) => h !== health)
            })
        });
      }
    }

    if (filters?.version?.source?.length) {
      for (const source of filters.version.source) {
        appliedFilters.push({
          key: 'source',
          value: source,
          onClear: () =>
            filterContext?.onFilter({
              ...filterContext?.filters,
              version: {
                ...filterContext?.filters?.version,
                source: filterContext?.filters?.version?.source?.filter((s) => s !== source)
              }
            })
        });
      }
    }

    if (filters?.version?.version?.length) {
      for (const version of filters.version.version) {
        appliedFilters.push({
          key: 'version',
          value: version,
          onClear: () =>
            filterContext?.onFilter({
              ...filterContext?.filters,
              version: {
                ...filterContext?.filters?.version,
                version: filterContext?.filters?.version?.version?.filter((v) => v !== version)
              }
            })
        });
      }
    }

    if (filters?.lastPromotion?.length) {
      appliedFilters.push({
        key: 'time',
        value: `${format(filters.lastPromotion[0], 'LLL co, yy')} - ${format(filters.lastPromotion[1], 'LLL co, yy')}`,
        onClear: () =>
          filterContext?.onFilter({ ...filterContext?.filters, lastPromotion: undefined })
      });
    }

    return appliedFilters;
  }, [filterContext?.filters, filterContext?.onFilter]);
};
