import { graphlib } from '@dagrejs/dagre';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { stackedIndexer, stageIndexer } from './node-indexer';
import { stackedLabelling } from './node-labeling';
import { stackSizer } from './node-sizer';

export const stackNodes = (
  afterNodes: string[],
  graph: graphlib.Graph,
  stageByName: Record<string, Stage>
) => {
  if (afterNodes.length === 0) {
    return {
      stackNodes: [],
      ignoreList: new Set<string>(),
      graph
    };
  }

  const sources = graph.sources();

  const stackNodes: Array<{
    count: number;
    parentNode: string;
    actualNode: string;
  }> = [];

  const ignoreList = new Set<string>();
  const processedParents = new Set<string>();
  const visited = new Set<string>();

  // @ts-expect-error it is string array
  const traverseQueue: string[] = [...sources];

  while (traverseQueue.length > 0) {
    const currentNode = traverseQueue.shift();

    if (currentNode && !visited.has(currentNode)) {
      visited.add(currentNode);

      const currentNodeSuccessors = graph.successors(currentNode) || [];

      for (const _successor of currentNodeSuccessors) {
        // @ts-expect-error type of successor is string
        const successor = _successor as string;

        if (afterNodes.includes(successor)) {
          if (processedParents.has(successor)) {
            continue;
          }
          processedParents.add(successor);

          const stackedNode: { count: number; parentNode: string; actualNode: string } = {
            count: 0,
            parentNode: successor,
            // @ts-expect-error it is string
            actualNode: graph.successors(successor)?.[0] as string
          };

          if (!stackedNode.actualNode) {
            continue;
          }

          const successors = getAllSuccessors(successor, graph);

          stackedNode.count = successors.size;

          for (const s of successors) {
            if (stackedIndexer.is(s)) {
              continue;
            }
            graph.removeNode(s);
            ignoreList.add(s);
          }

          stackNodes.push(stackedNode);

          const index = stackedIndexer.index(successor);

          graph.setNode(index, {
            ...stackedLabelling.label(
              stageByName[stageIndexer.getStageName(successor)],
              successor,
              stackedNode.count
            ),
            ...stackSizer.size()
          });

          graph.setEdge(successor, index);

          continue;
        }

        if (!visited.has(successor)) {
          traverseQueue.push(successor);
        }
      }
    }
  }

  return {
    stackNodes,
    ignoreList,
    graph
  };
};

const getAllSuccessors = (afterNode: string, graph: graphlib.Graph) => {
  const successors = new Set<string>();

  const traverseQueue: string[] = [afterNode];

  while (traverseQueue.length > 0) {
    const currentNode = traverseQueue.shift();

    if (currentNode) {
      for (const _successor of graph.successors(currentNode) || []) {
        // @ts-expect-error type of successor is string
        const successor = _successor as string;

        successors.add(successor);

        traverseQueue.push(successor);
      }
    }
  }

  return successors;
};
