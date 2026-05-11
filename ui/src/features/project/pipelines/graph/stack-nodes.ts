import { graphlib } from '@dagrejs/dagre';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { stackedIndexer, stageIndexer } from './node-indexer';
import { stackedLabelling } from './node-labeling';
import { stackSizer } from './node-sizer';

export const stackNodes = (
  afterNodes: string[],
  graph: graphlib.Graph,
  stageByName: Record<string, Stage>,
  maxStageHeight: number
): graphlib.Graph => {
  if (afterNodes.length === 0) {
    return graph;
  }

  const processedParents = new Set<string>();
  const visited = new Set<string>();

  // @ts-expect-error it is string array
  const traverseQueue: string[] = [...graph.sources()];

  while (traverseQueue.length > 0) {
    const currentNode = traverseQueue.shift();

    if (currentNode && !visited.has(currentNode)) {
      visited.add(currentNode);

      for (const successor of (graph.successors(currentNode) || []) as string[]) {
        if (afterNodes.includes(successor)) {
          if (processedParents.has(successor)) {
            continue;
          }
          processedParents.add(successor);

          const actualNode = graph.successors(successor)?.[0] as string;

          if (!actualNode) {
            continue;
          }

          const successors = getAllSuccessors(successor, graph);

          for (const s of successors) {
            if (!stackedIndexer.is(s)) {
              graph.removeNode(s);
            }
          }

          const index = stackedIndexer.index(successor);

          graph.setNode(index, {
            ...stackedLabelling.label(
              stageByName[stageIndexer.getStageName(successor)],
              successor,
              successors.size
            ),
            ...stackSizer.size(),
            height: maxStageHeight
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

  return graph;
};

const getAllSuccessors = (afterNode: string, graph: graphlib.Graph) => {
  const successors = new Set<string>();

  const traverseQueue: string[] = [afterNode];

  while (traverseQueue.length > 0) {
    const currentNode = traverseQueue.shift();

    if (currentNode) {
      for (const _successor of graph.successors(currentNode) || []) {
        const successor = _successor as string;

        successors.add(successor);

        traverseQueue.push(successor);
      }
    }
  }

  return successors;
};
