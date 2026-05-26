import { Label } from '@dagrejs/dagre';

import { WarehouseExpanded } from '@ui/extend/types';
import { RepoSubscription } from '@ui/gen/api/v1alpha1/generated_pb';
import { Stage } from '@ui/gen/api/v2/models';

export const warehouseLabelling = {
  label: (warehouse: WarehouseExpanded): Label => ({ warehouse })
};

export const repoSubscriptionLabelling = {
  label: (warehouse: WarehouseExpanded, subscription: RepoSubscription): Label => ({
    subscription,
    subscriptionParent: warehouse
  })
};

export const stageLabelling = {
  label: (stage: Stage): Label => ({ stage })
};

export const stackedLabelling = {
  label: (parentStage: Stage, parentStageId: string, count: number): Label => ({
    value: count,
    id: parentStage?.spec?.requestedFreight?.[0]?.origin?.name,
    parentNodeId: parentStageId
  })
};
