import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { layoutGraph } from './graph/layout-graph';
import { warehouseIndexer } from './graph/node-indexer';

export const groupNodes = (stages: Stage[], warehouses: Warehouse[]) => {
  const graph = layoutGraph({ stages: stages }, { warehouses: warehouses });

  const stackNodesAfter = new Set<string>();
  for (const warehouse of warehouses) {
    // after 5 nodes from this warehouse
    let node = warehouseIndexer.index(warehouse);
    let i = 0;
    while (i < 5) {
      const next = graph.successors(node);

      if (!next) {
        break;
      }

      // @ts-expect-error this is string
      node = next;
      i++;
    }

    if (Array.isArray(node)) {
      stackNodesAfter.add(node[0]);
    } else {
      stackNodesAfter.add(node);
    }
  }

  return [...stackNodesAfter];
};
