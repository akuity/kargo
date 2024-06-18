import { graphlib } from 'dagre';

import { RepoSubscription, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { AnyNodeType, NodeType, RepoNodeType } from '../types';

export const LINE_THICKNESS = 2;

export const NODE_WIDTH = 170;
export const NODE_HEIGHT = 130;

export const WAREHOUSE_NODE_WIDTH = 185;
export const WAREHOUSE_NODE_HEIGHT = 110;

export const initNodeArray = (s: Stage) =>
  [
    {
      data: s,
      type: NodeType.STAGE,
      color: '#000'
    }
  ] as AnyNodeType[];

export const getNodeType = (sub: RepoSubscription) =>
  sub.chart ? NodeType.REPO_CHART : sub.image ? NodeType.REPO_IMAGE : NodeType.REPO_GIT;

export const newSubscriptionNode = (
  sub: RepoSubscription,
  warehouseName: string
): RepoNodeType => ({
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: sub.chart || sub.image || sub.git || ({} as any),
  // stageNames: [stage?.metadata?.name || ''],
  warehouseName,
  type: getNodeType(sub)
});

export const nodeStubFor = (type: NodeType) => {
  const isStage = type === NodeType.STAGE;
  return {
    width: isStage ? NODE_WIDTH : WAREHOUSE_NODE_WIDTH,
    height: isStage ? NODE_HEIGHT : WAREHOUSE_NODE_HEIGHT
  };
};

export const getConnectors = (g: graphlib.Graph) => {
  return g.edges().map((item) => {
    const edge = g.edge(item);
    const points = edge.points;
    if (points.length > 0) {
      // replace first point with the right side of the upstream node
      const upstreamNode = g.node(item.v);
      if (upstreamNode) {
        points[0] = { x: upstreamNode.x + upstreamNode.width / 2, y: upstreamNode.y };
      }
    }
    if (points.length > 1) {
      // replace last point with the right side of the downstream node
      const upstreamNode = g.node(item.w);
      if (upstreamNode) {
        points[points.length - 1] = {
          x: upstreamNode.x - upstreamNode.width / 2,
          y: upstreamNode.y
        };
      }
    }

    const lines = new Array<{ x: number; y: number; width: number; angle: number }>();
    for (let i = 0; i < points.length - 1; i++) {
      const start = points[i];
      const end = points[i + 1];
      const x1 = start.x;
      const y1 = start.y;
      const x2 = end.x;
      const y2 = end.y;

      const width = Math.sqrt((x2 - x1) * (x2 - x1) + (y2 - y1) * (y2 - y1));
      // center
      const cx = (x1 + x2) / 2 - width / 2;
      const cy = (y1 + y2) / 2 - LINE_THICKNESS / 2;

      const angle = Math.atan2(y1 - y2, x1 - x2) * (180 / Math.PI);
      lines.push({ x: cx, y: cy, width, angle });
    }
    return lines;
  });
};
