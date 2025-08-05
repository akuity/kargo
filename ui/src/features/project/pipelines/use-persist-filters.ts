import { useEffect, useRef } from 'react';

import { FreightTimelineControllerContextType } from './context/freight-timeline-controller-context';

export const usePersistPreferredFilter = (
  project: string,
  preferredFilter: FreightTimelineControllerContextType['preferredFilter']
) => {
  const init = useRef(false);

  useEffect(() => {
    if (!init.current) {
      init.current = true;
      return;
    }

    localStorage.setItem(`filters-${project}`, JSON.stringify(preferredFilter));
  }, [preferredFilter]);
};

export const getFreightTimelineFiltersLocalStorage = (
  project?: string
): Partial<FreightTimelineControllerContextType['preferredFilter']> => {
  const filters = localStorage.getItem(`filters-${project}`);

  if (filters) {
    return JSON.parse(filters);
  }

  return {};
};
