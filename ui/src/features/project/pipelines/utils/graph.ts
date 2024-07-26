import { graphlib } from 'dagre';

import { RepoSubscription, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { AnyNodeType, ConnectorsType, NodeType, RepoNodeType } from '../types';

export const LINE_THICKNESS = 2;

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

export const getConnectors = (g: graphlib.Graph) => {
  const forward: { [key: string]: { [key: string]: ConnectorsType[][] } } = {};
  const backward: { [key: string]: { [key: string]: boolean } } = {};

  // horizontal edges are only between nodes where the parent has only one child and the child has only one parent
  g.edges().map((item) => {
    const edge = g.edge(item);
    const points = edge.points;

    const parts = item.name?.split(' ') || [];
    const from = parts[0] || '';
    const to = parts[1] || '';

    const lines = new Array<ConnectorsType>();
    for (let i = 0; i < points.length - 1; i++) {
      const start = points[i];
      const end = points[i + 1];
      const x1 = start.x;
      const y1 = start.y;
      const x2 = end.x;
      const y2 = end.y;

      const width = Math.sqrt((x2 - x1) * (x2 - x1) + (y2 - y1) * (y2 - y1)) + 2;
      // center
      const cx = (x1 + x2) / 2 - width / 2;
      const cy = (y1 + y2) / 2 - LINE_THICKNESS / 2;

      const angle = Math.atan2(y1 - y2, x1 - x2) * (180 / Math.PI);
      lines.push({ x: cx, y: cy, width, angle, color: edge['color'] });
    }

    const fromGr = forward[from] || {};
    forward[from] = { ...fromGr, [to]: [...(fromGr[to] || []), lines] };

    const backwardGr = backward[to] || {};
    backward[to] = { ...backwardGr, [from]: true };
  });

  for (const fromKey in forward) {
    if (Object.keys(forward[fromKey] || {}).length === 1) {
      for (const toKey of Object.keys(forward[fromKey])) {
        if (Object.keys(backward[toKey] || {}).length === 1) {
          const group = forward[fromKey][toKey];
          group.forEach((lines) => {
            lines.forEach((line) => {
              line.angle = 0;
            });
          });
        }
      }
    }
  }
  return Object.values(forward).flatMap((group) =>
    Object.values(group).flatMap((item) => Object.values(item))
  );
};
