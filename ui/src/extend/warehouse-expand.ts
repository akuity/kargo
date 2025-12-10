import { RepoSubscription, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import { WarehouseExpanded } from './types';

export const warehouseExpand = (warehouse: Warehouse): WarehouseExpanded => ({
  ...warehouse,
  spec: {
    ...warehouse?.spec,
    subscriptions: warehouse?.spec?.subscriptions?.map((s) => {
      let parsed = JSON.parse(
        decodeRawData({ result: { case: 'raw', value: s?.raw } })
      ) as RepoSubscription;

      parsed = {
        ...parsed,
        $typeName: 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription'
      };

      if (parsed.git) {
        parsed = {
          ...parsed,
          git: {
            ...parsed.git,
            $typeName: 'github.com.akuity.kargo.api.v1alpha1.GitSubscription'
          }
        };
      }

      if (parsed.image) {
        parsed = {
          ...parsed,
          image: {
            ...parsed.image,
            $typeName: 'github.com.akuity.kargo.api.v1alpha1.ImageSubscription'
          }
        };
      }

      if (parsed.chart) {
        parsed = {
          ...parsed,
          chart: {
            ...parsed.chart,
            $typeName: 'github.com.akuity.kargo.api.v1alpha1.ChartSubscription'
          }
        };
      }

      return parsed;
    })
  }
});
