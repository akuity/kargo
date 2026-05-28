import { useMemo } from 'react';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { FreightTimelineControllerContextType } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { timerangeToDate } from './filter-timerange-utils';
import { filterFreightBySource, filterFreightByTimerange } from './source-catalogue-utils';

export type FilteredFreight = Freight & { count?: number };

export const useFilteredFreights = (
  freights: Freight[],
  preferredFilter: FreightTimelineControllerContextType['preferredFilter']
): FilteredFreight[] => {
  const dictionaryContext = useDictionaryContext();
  const actionContext = useActionContext();

  const isPromotionMode =
    actionContext?.action?.type === IAction.PROMOTE ||
    actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM;

  return useMemo(() => {
    let filtered = [...(freights || [])].sort((a, b) => {
      const t1 = timestampDate(a?.metadata?.creationTimestamp);
      const t2 = timestampDate(b?.metadata?.creationTimestamp);
      return (t2?.getTime() || 0) - (t1?.getTime() || 0);
    });

    filtered = filtered
      .map(filterFreightBySource(preferredFilter.sources))
      .filter(Boolean) as Freight[];

    if (preferredFilter.timerange !== 'all-time') {
      filtered = filtered.filter(
        filterFreightByTimerange(timerangeToDate(preferredFilter.timerange))
      );
    }

    if (preferredFilter.warehouses?.length > 0) {
      filtered = filtered.filter((f) => preferredFilter.warehouses.includes(f.origin?.name || ''));
    }

    if (preferredFilter.hideUnusedFreights) {
      const collapsed: FilteredFreight[] = [];
      let count = 0;

      for (const f of filtered) {
        const inUse =
          (dictionaryContext?.freightInStages[f?.metadata?.name || '']?.length || 0) > 0;

        if (inUse) {
          if (count > 0) {
            collapsed.push({ ...f, count });
            count = 0;
          }
          collapsed.push(f);
        } else {
          count++;
        }
      }

      filtered = collapsed;
    }

    if (isPromotionMode) {
      filtered = filtered.filter((f) =>
        actionContext?.action?.stage?.spec?.requestedFreight?.find(
          (fr) => fr.origin?.name === f?.origin?.name
        )
      );
    }

    return filtered;
  }, [
    freights,
    preferredFilter.sources,
    preferredFilter.timerange,
    preferredFilter.warehouses,
    preferredFilter.hideUnusedFreights,
    dictionaryContext?.freightInStages,
    isPromotionMode,
    actionContext
  ]);
};
