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
      data: string;
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
  return {
    data: name,
    stageNames: stageNames || [],
    warehouseName: name,
    refreshing: !!warehouse?.metadata?.annotations['kargo.akuity.io/refresh'],
    type: NodeType.WAREHOUSE
  };
};
