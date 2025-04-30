import { useMemo } from 'react';

import { FreightList } from '@ui/gen/api/service/v1alpha1/service_pb';

export const useGetFreight = (freights: FreightList, search?: string) =>
  useMemo(
    () => !!search && freights?.freight?.find((f) => f?.metadata?.name === search),
    [freights, search]
  );
