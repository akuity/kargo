import { useMemo } from 'react';

import { Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

export const useGetWarehouse = (warehouses: Warehouse[], search?: string) =>
  useMemo(
    () => !!search && warehouses?.find((w) => w?.metadata?.name === search),
    [warehouses, search]
  );
