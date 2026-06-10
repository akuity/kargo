import { useMemo } from 'react';

import { useQueryFreightsRest } from '@ui/gen/api/v2/core/core';
import { Freight } from '@ui/gen/api/v2/models';

export const useGetFreightMap = (project: string) => {
  const freightQuery = useQueryFreightsRest(project);

  return useMemo(() => {
    const freightData = freightQuery.data?.data;

    // generate metadata.name -> full freight data (because history doesn't have it all) to show in freight history
    const freightMap: Record<string, Freight> = {};

    for (const freight of freightData?.groups?.['']?.items || []) {
      const freightId = freight?.metadata?.name;
      if (freightId) {
        freightMap[freightId] = freight;
      }
    }

    return freightMap;
  }, [freightQuery.data]);
};
