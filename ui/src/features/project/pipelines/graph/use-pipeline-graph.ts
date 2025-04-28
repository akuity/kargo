import { Edge, MarkerType, Node } from '@xyflow/react';
import { layout } from 'dagre';
import { useMemo } from 'react';

import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { edgeIndexer } from './edge-indexer';
import { layoutGraph } from './layout-graph';
import { warehouseIndexer } from './node-indexer';
import { stackNode } from './stack-nodes';

export const reactFlowNodeConstants = {
  CUSTOM_NODE: 'custom-node',
  STACKED_NODE: 'stacked-node'
};

export type GraphMeta = {
  warehouse?: Warehouse;
  subscription?: RepoSubscription;
  stage?: Stage;
  subscriptionParent?: Warehouse;
};

export const useReactFlowPipelineGraph = (
  stages: Stage[],
  warehouses: Warehouse[],
  // basically list of warehouses
  pipeline: string[],
  stack?: {
    afterNodes?: string[];
  }
) => {
  return useMemo(() => {
    const graph = layoutGraph(
      {
        stages,
        ignore(s) {
          return (
            !!pipeline.length &&
            !s.spec?.requestedFreight?.find((f) => pipeline.includes(f?.origin?.name || ''))
          );
        }
      },
      {
        warehouses,
        ignore(w) {
          return !!pipeline.length && !pipeline.includes(w?.metadata?.name || '');
        }
      }
    );

    layout(graph, { lablepos: 'c' });

    const reactFlowNodes: Node[] = [];
    const reactFlowEdges: Edge[] = [];

    const ignoreNodes = new Set<string>();
    const keepNodeAsStackNode: string[] = [];

    for (const afterNode of stack?.afterNodes || []) {
      const subIgnoreNodes = stackNode(afterNode, graph);

      if (subIgnoreNodes?.length > 0) {
        // in same pipeline
        if (!ignoreNodes.has(subIgnoreNodes[0])) {
          keepNodeAsStackNode.push(subIgnoreNodes[0]);
        }

        for (const inds of subIgnoreNodes) {
          ignoreNodes.add(inds);
        }
      }
    }

    for (const node of graph.nodes()) {
      const isStackNode = keepNodeAsStackNode.includes(node);
      const dagreNode = graph.node(node);

      if (isStackNode) {
        reactFlowNodes.push({
          id: node,
          type: reactFlowNodeConstants.STACKED_NODE,
          position: {
            x: dagreNode?.x - dagreNode?.width / 2,
            y: dagreNode?.y - dagreNode?.height / 2
          },
          data: {
            value: ignoreNodes.size,
            id: dagreNode?.stage?.spec?.requestedFreight?.[0]?.origin?.name,
            parentNodeId: graph.predecessors(node)?.[0]
          }
        });
        continue;
      }

      if (ignoreNodes.has(node)) {
        continue;
      }

      reactFlowNodes.push({
        id: node,
        type: reactFlowNodeConstants.CUSTOM_NODE,
        position: {
          x: dagreNode?.x - dagreNode?.width / 2,
          y: dagreNode?.y - dagreNode?.height / 2
        },
        data: {
          label: node,
          value: dagreNode?.warehouse || dagreNode?.subscription || dagreNode?.stage,
          subscriptionParent: dagreNode?.subscriptionParent
        }
      });
    }

    for (const edge of graph.edges()) {
      if (!keepNodeAsStackNode.includes(edge.v) && ignoreNodes.has(edge.v)) {
        continue;
      }

      const belongsToWarehouse = warehouseIndexer.getWarehouseName(edge.name || '');

      reactFlowEdges.push({
        id: edgeIndexer.index(belongsToWarehouse, edge.v, edge.w),
        source: edge.v,
        target: edge.w,
        animated: false,
        sourceHandle: belongsToWarehouse,
        targetHandle: belongsToWarehouse,
        markerEnd: {
          type: MarkerType.ArrowClosed
        },
        style: {
          strokeWidth: 2
        }
      });
    }

    return {
      nodes: reactFlowNodes,
      edges: reactFlowEdges
    };
  }, [stack?.afterNodes, pipeline]);
};
