import { graphlib } from 'dagre';

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
