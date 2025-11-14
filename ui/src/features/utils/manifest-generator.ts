import { stringify } from 'yaml';

import {
  ClusterPromotionTask,
  PromotionTask,
  WarehouseSpec
} from '@ui/gen/api/v1alpha1/generated_pb';
import { PartialRecursive, PlainMessageRecursive } from '@ui/utils/connectrpc-utils';
import { cleanEmptyObjectValues } from '@ui/utils/helpers';

// generate manifests for kargo resources
export const warehouseManifestsGen = {
  v1alpha1: (def: {
    projectName: string;
    warehouseName: string;
    spec: PartialRecursive<PlainMessageRecursive<WarehouseSpec>>;
  }) =>
    stringify({
      apiVersion: 'kargo.akuity.io/v1alpha1',
      kind: 'Warehouse',
      metadata: {
        name: def.warehouseName,
        namespace: def.projectName
      },
      spec: cleanEmptyObjectValues(def.spec)
    })
};

export const promotionTaskManifestsGen = {
  v1alpha1: (def: PromotionTask) =>
    stringify({
      apiVersion: 'kargo.akuity.io/v1alpha1',
      kind: 'PromotionTask',
      ...def
    })
};

export const clusterPromotionTaskManifestsGen = {
  v1alpha1: (def: ClusterPromotionTask) =>
    stringify({
      apiVersion: 'kargo.akuity.io/v1alpha1',
      kind: 'ClusterPromotionTask',
      ...def
    })
};
