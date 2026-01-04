import { WarehouseExpanded } from '@ui/extend/types';
import { RepoSubscription, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const warehouseIndexer = {
  index: (warehouse: WarehouseExpanded) => {
    const warehouseUID = warehouse?.metadata?.uid;
    const warehouseName = warehouse?.metadata?.name;

    return `warehouse/${warehouseUID}/${warehouseName}`;
  },
  getWarehouseName: (index: string) => index.split('/')[2]
};

export const repoSubscriptionIndexer = {
  index: (wh: WarehouseExpanded, subscription: RepoSubscription) => {
    const warehouseIndex = warehouseIndexer.index(wh);
    const subscriptionRepoURL =
      subscription?.image?.repoURL ||
      subscription?.git?.repoURL ||
      subscription?.chart?.repoURL ||
      `${subscription?.subscription?.name}${subscription?.subscription?.subscriptionType}` ||
      'unknown';

    return `subscription/${warehouseIndex}/${subscriptionRepoURL}`;
  },
  is: (id: string) => id.startsWith('subscription/')
};

export const stageIndexer = {
  index: (stage: Stage) => {
    const stageUID = stage?.metadata?.uid;
    const stageName = stage?.metadata?.name;

    return `stage/${stageUID}/${stageName}`;
  },
  getStageName: (index: string) => index.split('/')[2]
};

export const stackedIndexer = {
  index: (parentNode: string) => `stacked/${parentNode}`,
  is: (id: string) => id.startsWith('stacked/')
};
