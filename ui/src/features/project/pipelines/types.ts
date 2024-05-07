import {
  ChartSubscription,
  GitSubscription,
  ImageSubscription,
  Stage,
  Warehouse
} from '@ui/gen/v1alpha1/generated_pb';

export enum NodeType {
  STAGE,
  REPO_IMAGE,
  REPO_GIT,
  REPO_CHART,
  WAREHOUSE
}

type NodeBase = {
  stageNames: string[];
  warehouseName: string;
  refreshing?: boolean;
};

export type NodesRepoType = (
  | {
      type: NodeType.REPO_IMAGE;
      data: ImageSubscription;
    }
  | {
      type: NodeType.REPO_GIT;
      data: GitSubscription;
    }
  | {
      type: NodeType.REPO_CHART;
      data: ChartSubscription;
    }
  | {
      type: NodeType.WAREHOUSE;
      data: Warehouse;
    }
) &
  NodeBase;

export type NodesItemType =
  | {
      type: NodeType.STAGE;
      data: Stage;
      color: string;
    }
  | NodesRepoType;

export const NewWarehouseNode = (warehouse: Warehouse, stageNames?: string[]): NodesRepoType => {
  const name = warehouse?.metadata?.name || '';
  const refreshRequest = warehouse?.metadata?.annotations['kargo.akuity.io/refresh'];
  const refreshStatus = warehouse?.status?.lastHandledRefresh;
  return {
    data: warehouse,
    stageNames: stageNames || [],
    warehouseName: name,
    refreshing: refreshRequest !== undefined && refreshRequest !== refreshStatus,
    type: NodeType.WAREHOUSE
  };
};

export type FreightlineAction = 'promote' | 'promoteSubscribers' | 'manualApproval';

export enum FreightMode {
  Default = 'default', // not promoting, has stages
  Promotable = 'promotable', // promoting, promotable
  Disabled = 'disabled',
  Selected = 'selected',
  Confirming = 'confirming' // promoting, confirming
}
