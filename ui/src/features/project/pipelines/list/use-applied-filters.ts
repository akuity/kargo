import { useMemo } from 'react';

import { useFilterContext } from './context/filter-context';

type AppliedFilter = {
  key: string;
  value: string;
};

export const useAppliedFilters = () => {
  const filterContext = useFilterContext();

  return useMemo(() => {
    const filters = filterContext?.filters;

    const appliedFilters: AppliedFilter[] = [];

    if (filters?.stage) {
      appliedFilters.push({
        key: 'stage',
        value: filters.stage
      });
    }

    if (filters?.phase?.length) {
      for (const phase of filters.phase) {
        appliedFilters.push({
          key: 'phase',
          value: phase
        });
      }
    }

    if (filters?.health?.length) {
      for (const health of filters.health) {
        appliedFilters.push({
          key: 'health',
          value: health
        });
      }
    }

    if (filters?.version?.source?.length) {
      for (const source of filters.version.source) {
        appliedFilters.push({
          key: 'source',
          value: source
        });
      }
    }

    if (filters?.version?.version?.length) {
      for (const version of filters.version.version) {
        appliedFilters.push({
          key: 'version',
          value: version
        });
      }
    }

    if (filters?.lastPromotion?.length) {
      appliedFilters.push({
        key: 'start time',
        value: filters.lastPromotion[0].toString()
      });

      appliedFilters.push({
        key: 'end time',
        value: filters.lastPromotion[1].toString()
      });
    }

    return appliedFilters;
  }, [filterContext?.filters]);
};
