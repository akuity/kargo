import { Label } from 'dagre';

import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

export const warehouseLabelling = {
  label: (warehouse: Warehouse): Label => ({ warehouse })
};

export const repoSubscriptionLabelling = {
  label: (warehouse: Warehouse, subscription: RepoSubscription): Label => ({
    subscription,
    subscriptionParent: warehouse
  })
};

export const stageLabelling = {
  label: (stage: Stage): Label => ({ stage })
};
