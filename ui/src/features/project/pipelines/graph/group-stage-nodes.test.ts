import { graphlib, layout } from '@dagrejs/dagre';
import { describe, expect, test } from 'vitest';

import { Stage } from '@ui/gen/api/v2/models';

import { groupStages, STAGE_GROUP_LABEL } from '../stage-groups';

import { groupStageNodes, uncollapsibleStageGroups } from './group-stage-nodes';
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

// warehouse -> dev-1, dev-2, systest; dev-1 and dev-2 -> after-dev
const build = () => {
  const g = new graphlib.Graph({ multigraph: true });
  g.setGraph({ rankdir: 'LR' });
  g.setDefaultEdgeLabel(() => ({}));
  for (const s of all) g.setNode(stageIndexer.index(s), { stage: s });
  g.setNode('warehouse/w/main', {});
  for (const s of [dev1, dev2, gate])
    g.setEdge('warehouse/w/main', stageIndexer.index(s), {}, 'main');
  for (const s of [dev1, dev2])
    g.setEdge(stageIndexer.index(s), stageIndexer.index(after), {}, 'main');
  return g;
};

// warehouse -> dev -> qa -> prod
const chain = (groupOf: Record<string, string>) => {
  const stages = ['dev', 'qa', 'prod'].map((n) => stage(n, groupOf[n]));
  const g = new graphlib.Graph({ multigraph: true });
  g.setDefaultEdgeLabel(() => ({}));
  g.setNode('warehouse/w/main', {});
  for (const s of stages) g.setNode(stageIndexer.index(s), { stage: s });
  const ids = ['warehouse/w/main', ...stages.map((s) => stageIndexer.index(s))];
  for (let i = 1; i < ids.length; i++) g.setEdge(ids[i - 1], ids[i], {}, 'main');
  return { g, stages };
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

  test('takes over the shared incoming and downstream edges', () => {
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

  test('dev and prod of warehouse -> dev -> qa -> prod are never collapsed', () => {
    const { g, stages } = chain({ dev: 'g', prod: 'g' });
    const groups = groupStages(stages);
    expect([...uncollapsibleStageGroups(g, groups)]).toEqual(['g']);
    const out = groupStageNodes(g, groups, 200);
    expect(out.hasNode(stageGroupIndexer.index('g'))).toBe(false);
    expect(out.edgeCount()).toBe(3);
  });

  test('members wired to each other are never collapsed', () => {
    const { g, stages } = chain({ dev: 'g', qa: 'g' });
    expect([...uncollapsibleStageGroups(g, groupStages(stages))]).toEqual(['g']);
  });

  test('members with different downstream stages are never collapsed', () => {
    const g = build();
    g.removeEdge(stageIndexer.index(dev2), stageIndexer.index(after), 'main');
    expect([...uncollapsibleStageGroups(g, groupStages(all))]).toEqual(['dev']);
  });

  test('members with different upstream sources are never collapsed', () => {
    const g = build();
    g.setNode('warehouse/w/other', {});
    g.setEdge('warehouse/w/other', stageIndexer.index(dev1), {}, 'other');
    expect([...uncollapsibleStageGroups(g, groupStages(all))]).toEqual(['dev']);
  });
});
