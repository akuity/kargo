import { useMemo } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';

export const useGetWarehouse = (warehouses: WarehouseExpanded[], search?: string) =>
  useMemo(
    () => !!search && warehouses?.find((w) => w?.metadata?.name === search),
    [warehouses, search]
  );
