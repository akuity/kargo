import { RepoSubscription, Subscription, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import { WarehouseExpanded } from './types';

export const warehouseExpand = (warehouse: Warehouse): WarehouseExpanded => ({
  ...warehouse,
  spec: {
    ...warehouse?.spec,
    subscriptions: warehouse?.spec?.subscriptions?.map((s) => {
      const parsed = JSON.parse(
        decodeRawData({ result: { case: 'raw', value: s?.raw } })
      ) as RepoSubscription & { [key: string]: Subscription };

      if (parsed.git) {
        return {
          $typeName: 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription',
          git: {
            ...parsed.git,
            $typeName: 'github.com.akuity.kargo.api.v1alpha1.GitSubscription'
          }
        };
      }

      if (parsed.image) {
        return {
          $typeName: 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription',
          image: {
            ...parsed.image,
            $typeName: 'github.com.akuity.kargo.api.v1alpha1.ImageSubscription'
          }
        };
      }

      if (parsed.chart) {
        return {
          $typeName: 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription',
          chart: {
            ...parsed.chart,
            $typeName: 'github.com.akuity.kargo.api.v1alpha1.ChartSubscription'
          }
        };
      }

      const otherSubscriptionKey = Object.keys(parsed)[0];

      return {
        $typeName: 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription',
        subscription: parsed[otherSubscriptionKey]
      };
    })
  }
});
