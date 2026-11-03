import { RepoSubscription, WarehouseExpanded } from '@ui/extend/types';
import { Stage, Warehouse } from '@ui/gen/api/v2/models';

// Membership changes need a relayout even when no content-update callback is
// emitted (notably DELETED watch events). Status-only updates do not.
export const resourceMembership = (resources: Pick<Stage, 'metadata'>[]) =>
  JSON.stringify(
    resources.map((resource) => [resource.metadata?.uid, resource.metadata?.name]).sort()
  );

export const warehouseIndexer = {
  index: (warehouse: Pick<Warehouse, 'metadata'>) => {
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
      `${subscription?.name}${subscription?.subscription?.subscriptionType}` ||
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
