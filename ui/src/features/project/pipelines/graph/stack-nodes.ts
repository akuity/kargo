import { graphlib } from 'dagre';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { stackedIndexer, stageIndexer } from './node-indexer';
import { stackedLabelling } from './node-labeling';
import { stackSizer } from './node-sizer';

export const stackNode = (afterNode: string, graph: graphlib.Graph) => {
  const traverseQueue: string[] = [];

  if (afterNode) {
    traverseQueue.push(afterNode);
  }

  const ignoreNodes: string[] = [];
  while (traverseQueue.length > 0) {
    const firstNode = traverseQueue.shift();

    if (firstNode) {
      for (const successor of graph.successors(firstNode) || []) {
        // @ts-expect-error type of successor is string
        ignoreNodes.push(successor);
        // @ts-expect-error type of successor is string
        traverseQueue.push(successor);
      }
    }
  }

  return ignoreNodes;
};

export const stackNodes = (
  afterNodes: string[],
  graph: graphlib.Graph,
  stageByName: Record<string, Stage>
) => {
  const sources = graph.sources();

  const stackNodes: Array<{
    count: number;
    parentNode: string;
  }> = [];

  const ignoreList = new Set<string>();

  // @ts-expect-error it is string array
  const traverseQueue: string[] = [...sources];

  while (traverseQueue.length > 0) {
    const currentNode = traverseQueue.shift();

    if (currentNode) {
      for (const _successor of graph.successors(currentNode) || []) {
        // @ts-expect-error type of successor is string
        const successor = _successor as string;

        if (afterNodes.includes(successor)) {
          const stackedNode: { count: number; parentNode: string } = {
            count: 0,
            parentNode: successor
          };

          const successors = getAllSuccessors(successor, graph);

          stackedNode.count = successors.size;

          for (const s of successors) {
            if (stackedIndexer.is(s)) {
              continue;
            }
            graph.removeNode(s);
            ignoreList.add(s);
          }

          if (!stackNodes.find((s) => s.parentNode === successor)) {
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
          }

          continue;
        }

        traverseQueue.push(successor);
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
