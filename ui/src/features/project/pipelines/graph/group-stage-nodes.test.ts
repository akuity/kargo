import { graphlib, layout } from '@dagrejs/dagre';
import { describe, expect, test } from 'vitest';

import { Stage } from '@ui/gen/api/v2/models';

import { groupStages, STAGE_GROUP_LABEL } from '../stage-groups';

import { groupStageNodes } from './group-stage-nodes';
import { stageGroupIndexer, stageIndexer } from './node-indexer';

const stage = (name: string, group?: string): Stage =>
  ({
    metadata: { name, uid: name, labels: group ? { [STAGE_GROUP_LABEL]: group } : {} }
  }) as Stage;

const dev1 = stage('dev-1', 'dev');
const dev2 = stage('dev-2', 'dev');
const gate = stage('systest');
const after = stage('after-dev');
const all = [dev1, dev2, gate, after];

const build = () => {
  const g = new graphlib.Graph({ multigraph: true });
  g.setGraph({ rankdir: 'LR' });
  g.setDefaultEdgeLabel(() => ({}));
  for (const s of all) g.setNode(stageIndexer.index(s), { stage: s });
  g.setNode('warehouse/w/main', {});
  for (const s of [dev1, dev2, gate])
    g.setEdge('warehouse/w/main', stageIndexer.index(s), {}, 'main');
  g.setEdge(stageIndexer.index(dev1), stageIndexer.index(dev2), {}, 'main');
  g.setEdge(stageIndexer.index(dev2), stageIndexer.index(after), {}, 'main');
  return g;
};

describe('groupStageNodes', () => {
  test('replaces collapsed members with one node and keeps other stages', () => {
    const g = groupStageNodes(build(), groupStages(all), 200);
    const group = stageGroupIndexer.index('dev');
    expect(g.nodes().sort()).toEqual(
      [group, stageIndexer.index(gate), stageIndexer.index(after), 'warehouse/w/main'].sort()
    );
    expect(g.node(group).group.stages).toHaveLength(2);
  });

  test('merges incoming edges, drops edges between members, keeps downstream edge', () => {
    const g = groupStageNodes(build(), groupStages(all), 200);
    const group = stageGroupIndexer.index('dev');
    const edges = g.edges().map((e) => `${e.v}->${e.w}:${e.name}`);
    expect(edges.sort()).toEqual(
      [
        `warehouse/w/main->${group}:main`,
        `warehouse/w/main->${stageIndexer.index(gate)}:main`,
        `${group}->${stageIndexer.index(after)}:main`
      ].sort()
    );
  });

  test('expanded groups are left untouched', () => {
    const g = groupStageNodes(build(), groupStages(all, ['dev']), 200);
    expect(g.hasNode(stageGroupIndexer.index('dev'))).toBe(false);
    expect(g.hasNode(stageIndexer.index(dev1))).toBe(true);
  });

  test('the grouped graph lays out with dagre', () => {
    const g = groupStageNodes(build(), groupStages(all), 200);
    for (const n of g.nodes()) g.setNode(n, { ...g.node(n), width: 250, height: 100 });
    layout(g);
    expect(g.node(stageGroupIndexer.index('dev')).x).toBeGreaterThan(g.node('warehouse/w/main').x);
  });
});
