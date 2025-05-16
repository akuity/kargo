import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

export const warehouseIndexer = {
  index: (warehouse: Warehouse) => {
    const warehouseUID = warehouse?.metadata?.uid;
    const warehouseName = warehouse?.metadata?.name;

    return `warehouse/${warehouseUID}/${warehouseName}`;
  },
  getWarehouseName: (index: string) => index.split('/')[2]
};

export const repoSubscriptionIndexer = {
  index: (wh: Warehouse, subscription: RepoSubscription) => {
    const warehouseIndex = warehouseIndexer.index(wh);
    const subscriptionRepoURL =
      subscription?.image?.repoURL || subscription?.git?.repoURL || subscription?.chart?.repoURL;

    return `subscription/${warehouseIndex}/${subscriptionRepoURL}`;
  }
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
