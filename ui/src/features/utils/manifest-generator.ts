import { PlainMessage } from '@bufbuild/protobuf';
import { stringify } from 'yaml';

import { WarehouseSpec } from '@ui/gen/v1alpha1/generated_pb';
import { cleanEmptyObjectValues } from '@ui/utils/helpers';

// generate manifests for kargo resources
export const WarehouseManifestsGen = {
  v1alpha1: (def: {
    projectName: string;
    warehouseName: string;
    spec: PlainMessage<WarehouseSpec>;
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
