import { useQuery } from '@connectrpc/connect-query';
import { useMemo } from 'react';

import { queryFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

export const useGetFreightMap = (project: string) => {
  const freightQuery = useQuery(queryFreight, { project });

  return useMemo(() => {
    const freightData = freightQuery.data;

    // generate metadata.name -> full freight data (because history doesn't have it all) to show in freight history
    const freightMap: Record<string, Freight> = {};

    for (const freight of freightData?.groups?.['']?.freight || []) {
      const freightId = freight?.metadata?.name;
      if (freightId) {
        freightMap[freightId] = freight;
      }
    }

    return freightMap;
  }, [freightQuery.data]);
};
