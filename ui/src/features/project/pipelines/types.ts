import {
  ChartSubscription,
  GitSubscription,
  ImageSubscription,
  Stage,
  Warehouse
} from '@ui/gen/v1alpha1/generated_pb';

import { RepoNodeDimensions } from './nodes/repo-node';
import { StageNodeDimensions } from './nodes/stage-node';

export enum NodeType {
  STAGE,
  REPO_IMAGE,
  REPO_GIT,
  REPO_CHART,
  WAREHOUSE
}

export type NodeDimensions = {
  width: number;
  height: number;
};

export const getNodeDimensions = (type: NodeType): NodeDimensions => {
  switch (type) {
    case NodeType.STAGE:
      return StageNodeDimensions();
    default:
      return RepoNodeDimensions();
  }
};

type NodeBase = {
  stageNames?: string[];
  warehouseName: string;
  refreshing?: boolean;
};

export type RepoNodeType = (
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

export type AnyNodeType =
  | {
      type: NodeType.STAGE;
      data: Stage;
      color: string;
    }
  | RepoNodeType;

export const NewWarehouseNode = (warehouse: Warehouse, stageNames?: string[]): RepoNodeType => {
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

export enum FreightTimelineAction {
  Promote = 'promote', // Promoting a stage. Freight has not been selected yet
  PromoteSubscribers = 'promoteSubscribers', // Promoting subscribers of a stage. Freight has not been selected yet
  ManualApproval = 'manualApproval', // Manually approving a freight
  PromoteFreight = 'promoteFreight' // Promoting a specific freight to any stage
}

export enum FreightMode {
  Default = 'default', // not promoting, has stages
  Promotable = 'promotable', // promoting, promotable
  Disabled = 'disabled',
  Selected = 'selected',
  Confirming = 'confirming' // promoting, confirming
}

export interface ConnectorsType {
  x: number;
  y: number;
  width: number;
  angle: number;
  color: string;
  from: string;
  to: string;
}

export interface BoxType {
  width: number;
  height: number;
}

export type DagreNode = AnyNodeType & {
  top: number;
  left: number;
  width: number;
  height: number;
};

export interface StagePixelStyle {
  opacity: number;
  backgroundColor: string;
  border?: string;
}

export type StageStyleMap = { [key: string]: StagePixelStyle };
