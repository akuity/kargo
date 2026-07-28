import { useMemo } from 'react';

import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { useFreightTimelineControllerStore } from '@ui/features/project/pipelines/url-params/use-freight-timeline-controller-store';
import { useQueryFreightsRest } from '@ui/gen/api/v2/core/core';
import { Freight } from '@ui/gen/api/v2/models';

export const useGetFreightMap = (project: string) => {
  const dictionaryContext = useDictionaryContext();
  const [preferredFilter] = useFreightTimelineControllerStore(project);

  // Stage details is rendered by the pipeline view, which already loads the
  // project's Freight into DictionaryContext. Reuse that map instead of issuing
  // a duplicate queryFreight request. This is only safe when no warehouse filter
  // is active: a filtered context map omits Freight that Stage details still
  // needs to resolve (e.g. aliases in the promotions/verifications tables), so
  // in that case we fall back to our own unfiltered fetch.
  const canReuseContext = !!dictionaryContext && preferredFilter.warehouses.length === 0;

  const freightQuery = useQueryFreightsRest(project, {}, { query: { enabled: !canReuseContext } });

  return useMemo(() => {
    if (canReuseContext) {
      return dictionaryContext.freightById;
    }

    // generate metadata.name -> full freight data (because history doesn't have it all) to show in freight history
    const freightMap: Record<string, Freight> = {};

    for (const freight of freightQuery.data?.data.groups?.['']?.items || []) {
      const freightId = freight?.metadata?.name;
      if (freightId) {
        freightMap[freightId] = freight;
      }
    }

    return freightMap;
  }, [canReuseContext, dictionaryContext, freightQuery.data]);
};
