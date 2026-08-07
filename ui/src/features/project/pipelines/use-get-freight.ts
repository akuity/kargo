import { useMemo } from 'react';

import { Freight } from '@ui/gen/api/v2/models';

export const useGetFreight = (freights: Freight[], search?: string) =>
  useMemo(
    () => !!search && freights?.find((f) => f?.metadata?.name === search),
    [freights, search]
  );
