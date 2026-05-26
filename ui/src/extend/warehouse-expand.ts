import { Warehouse } from '@ui/gen/api/v2/models';

import { RepoSubscription, WarehouseExpanded } from './types';

export const warehouseExpand = (warehouse: Warehouse): WarehouseExpanded => ({
  ...warehouse,
  spec: {
    ...warehouse?.spec,
    subscriptions: warehouse?.spec?.subscriptions?.map((s: RepoSubscription) => {
      if (s.git) {
        return {
          git: {
            ...s.git
          }
        };
      }

      if (s.image) {
        return {
          image: {
            ...s.image
          }
        };
      }

      if (s.chart) {
        return {
          chart: {
            ...s.chart
          }
        };
      }

      return {
        subscription: s.subscription
      };
    })
  }
});
