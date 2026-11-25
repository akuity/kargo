import { Edge, graphlib } from '@dagrejs/dagre';

import { Stage } from '@ui/gen/api/v2/models';

import { StageGroups } from '../stage-groups';

import { stageIndexer, stageGroupIndexer } from './node-indexer';
import { stackSizer } from './node-sizer';

const edgeKeys = (edges: Edge[] | void, end: 'v' | 'w') =>
  (edges || [])
    .map((e) => `${e[end]}|${e.name ?? ''}`)
    .sort()
    .join('\n');

// A group can only be drawn as one node when that node's edges are true for every member:
// all members request Freight from the same sources (warehouses, upstream stages), the same
// stages request Freight from each of them, and no member feeds another. Otherwise a collapsed
// group would show members wired to stages they are not wired to, e.g. dev and prod of
// warehouse -> dev -> qa -> prod. Such groups are never collapsed.
export const uncollapsibleStageGroups = (
  graph: graphlib.Graph,
  groups: StageGroups
): Set<string> => {
  const blocked = new Set<string>();
  for (const [name, stages] of Object.entries(groups.members)) {
    const ids = stages.map((s) => stageIndexer.index(s)).filter((id) => graph.hasNode(id));
    const memberIds = new Set(ids);
    const linked = ids.some((id) =>
      (graph.successors(id) || []).some((w) => memberIds.has(w as string))
    );
    const inKeys = new Set(ids.map((id) => edgeKeys(graph.inEdges(id), 'v')));
    const outKeys = new Set(ids.map((id) => edgeKeys(graph.outEdges(id), 'w')));
    if (linked || inKeys.size > 1 || outKeys.size > 1) {
      blocked.add(name);
    }
  }
  return blocked;
};

// Replaces the stage nodes of every collapsed group with one `stage-group/<name>` node.
// Groups from uncollapsibleStageGroups are skipped. For the rest every member has the same
// incoming and outgoing edges, so the group node takes them over unchanged.
export const groupStageNodes = (
  graph: graphlib.Graph,
  groups: StageGroups,
  maxStageHeight: number
): graphlib.Graph => {
  const blocked = uncollapsibleStageGroups(graph, groups);
  for (const name of groups.collapsed) {
    if (blocked.has(name)) {
      continue;
    }
    const memberIds = new Set<string>();
    const memberStages: Stage[] = [];
    for (const stage of groups.members[name]) {
      const id = stageIndexer.index(stage);
      if (graph.hasNode(id)) {
        memberIds.add(id);
        memberStages.push(stage);
      }
    }
    if (memberIds.size === 0) {
      continue;
    }

    const groupId = stageGroupIndexer.index(name);
    graph.setNode(groupId, {
      group: { name, stages: memberStages },
      ...stackSizer.size(),
      height: maxStageHeight
    });

    for (const edge of [...(graph.edges() || [])]) {
      const fromMember = memberIds.has(edge.v);
      const toMember = memberIds.has(edge.w);
      if (!fromMember && !toMember) {
        continue;
      }
      const label = graph.edge(edge);
      if (!(fromMember && toMember)) {
        // members share their edges, so the copies overwrite each other into one edge
        graph.setEdge(fromMember ? groupId : edge.v, toMember ? groupId : edge.w, label, edge.name);
      }
    }

    for (const id of memberIds) {
      graph.removeNode(id);
    }
  }

  return graph;
};
