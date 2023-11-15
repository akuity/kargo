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

export type NodesRepoType =
  | {
      type: NodeType.REPO_IMAGE;
      data: ImageSubscription;
      stageName: string;
      warehouseName: string;
    }
  | {
      type: NodeType.REPO_GIT;
      data: GitSubscription;
      stageName: string;
      warehouseName: string;
    }
  | {
      type: NodeType.REPO_CHART;
      data: ChartSubscription;
      stageName: string;
      warehouseName: string;
    };

export type NodesItemType =
  | {
      type: NodeType.STAGE;
      data: Stage;
      color: string;
    }
  | NodesRepoType;
