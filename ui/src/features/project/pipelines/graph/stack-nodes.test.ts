import { graphlib } from '@dagrejs/dagre';
import { describe, expect, test, vi } from 'vitest';

import { Stage } from '@ui/gen/api/v2/models';

import { stackedIndexer } from './node-indexer';
import { stackNodes } from './stack-nodes';

const stageId = (name: string) => `stage/${name}-uid/${name}`;

const buildFanOutFanInGraph = () => {
  const graph = new graphlib.Graph();

  const edges: [string, string][] = [
    ['warehouse', 'parent'],
    ['parent', 'a1'],
    ['parent', 'a2'],
    ['a1', 'b'],
    ['a2', 'b'],
    ['b', 'c1'],
    ['b', 'c2'],
    ['c1', 'd'],
    ['c2', 'd'],
    ['d', 'e1'],
    ['d', 'e2'],
    ['e1', 'f'],
    ['e2', 'f']
  ];

  for (const [from, to] of edges) {
    const source = from === 'warehouse' ? from : stageId(from);
    graph.setNode(source, {});
    graph.setNode(stageId(to), {});
    graph.setEdge(source, stageId(to));
  }

  return graph;
};

const stageByName: Record<string, Stage> = {
  parent: {
    metadata: { name: 'parent' },
    spec: { requestedFreight: [{ origin: { kind: 'Warehouse', name: 'warehouse' } }] }
  } as Stage
};

describe('stackNodes', () => {
  test('expands each converged node once in fan-out/fan-in graphs', () => {
    const graph = buildFanOutFanInGraph();
    const successorsSpy = vi.spyOn(graph, 'successors');

    stackNodes([stageId('parent')], graph, stageByName, 200);

    const callsFor = (name: string) =>
      successorsSpy.mock.calls.filter(([node]) => node === stageId(name)).length;

    expect(callsFor('b')).toBe(1);
    expect(callsFor('d')).toBe(1);
    expect(callsFor('f')).toBe(1);
  });

  test('skips a queued node that stacking removes', () => {
    const graph = new graphlib.Graph();
    const parent = stageId('parent');
    const shared = stageId('shared');

    graph.setNode('source-a', {});
    graph.setNode('source-b', {});
    graph.setNode(parent, {});
    graph.setNode(shared, {});
    graph.setEdge('source-a', shared);
    graph.setEdge('source-b', parent);
    graph.setEdge(parent, shared);

    const successorsSpy = vi.spyOn(graph, 'successors');

    stackNodes([parent], graph, stageByName, 200);

    const sharedCalls = successorsSpy.mock.calls.filter(([node]) => node === shared);
    expect(sharedCalls).toHaveLength(1);
  });

  test('replaces all downstream nodes with a single stack of the correct count', () => {
    const graph = buildFanOutFanInGraph();
    const parent = stageId('parent');
    const stacked = stackedIndexer.index(parent);

    stackNodes([parent], graph, stageByName, 200);

    expect(graph.nodes().sort()).toEqual(['warehouse', parent, stacked].sort());
    expect(graph.successors(parent)).toEqual([stacked]);
    expect(graph.node(stacked)).toMatchObject({
      value: 9,
      id: 'warehouse',
      parentNodeId: parent,
      height: 200
    });
  });
});
