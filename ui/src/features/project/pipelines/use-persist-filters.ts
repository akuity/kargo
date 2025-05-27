import { useEffect, useRef } from 'react';

import { FreightTimelineControllerContextType } from './context/freight-timeline-controller-context';

export const usePersistPreferredFilter = (
  project: string,
  preferredFilter: FreightTimelineControllerContextType['preferredFilter'],
  updatePreferredFilter: (f: FreightTimelineControllerContextType['preferredFilter']) => void
) => {
  const init = useRef(false);

  useEffect(() => {
    if (!init.current) {
      init.current = true;
      return;
    }

    localStorage.setItem(`filters-${project}`, JSON.stringify(preferredFilter));
  }, [preferredFilter]);

  useEffect(() => {
    const preferredFilterLocal = localStorage.getItem(`filters-${project}`);

    if (preferredFilterLocal) {
      try {
        updatePreferredFilter(JSON.parse(preferredFilterLocal));
      } catch (e) {
        // silent
      }
    }
  }, []);
};
