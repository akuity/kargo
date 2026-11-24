import { graphlib } from '@dagrejs/dagre';

import { Stage } from '@ui/gen/api/v2/models';

import { StageGroups } from '../stage-groups';

import { stageIndexer, stageGroupIndexer } from './node-indexer';
import { stackSizer } from './node-sizer';

// Replaces the stage nodes of every collapsed group with one `stage-group/<name>` node.
// Incoming edges of the members (from warehouses or upstream stages) are merged onto the
// group node; edges between members vanish. Outgoing edges of a member (stages that request
// Freight from a grouped stage) start at the group node, so downstream stages still render.
export const groupStageNodes = (
  graph: graphlib.Graph,
  groups: StageGroups,
  maxStageHeight: number
): graphlib.Graph => {
  for (const name of groups.collapsed) {
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
        // same edge name (warehouse) + endpoints overwrite each other, which dedupes the union
        graph.setEdge(fromMember ? groupId : edge.v, toMember ? groupId : edge.w, label, edge.name);
      }
    }

    for (const id of memberIds) {
      graph.removeNode(id);
    }
  }

  return graph;
};
