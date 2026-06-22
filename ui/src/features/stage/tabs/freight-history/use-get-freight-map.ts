import { useQuery } from '@connectrpc/connect-query';
import { useMemo } from 'react';

import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { useFreightTimelineControllerStore } from '@ui/features/project/pipelines/url-params/use-freight-timeline-controller-store';
import { queryFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

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

  const freightQuery = useQuery(queryFreight, { project }, { enabled: !canReuseContext });

  return useMemo(() => {
    if (canReuseContext) {
      return dictionaryContext.freightById;
    }

    // generate metadata.name -> full freight data (because history doesn't have it all) to show in freight history
    const freightMap: Record<string, Freight> = {};

    for (const freight of freightQuery.data?.groups?.['']?.freight || []) {
      const freightId = freight?.metadata?.name;
      if (freightId) {
        freightMap[freightId] = freight;
      }
    }

    return freightMap;
  }, [canReuseContext, dictionaryContext, freightQuery.data]);
};
