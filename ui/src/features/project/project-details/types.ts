import {
  ChartSubscription,
  GitSubscription,
  ImageSubscription,
  Stage
} from '@ui/gen/v1alpha1/types_pb';

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
