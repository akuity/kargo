import { layout } from '@dagrejs/dagre';
import { Edge, MarkerType, Node } from '@xyflow/react';
import { useContext, useMemo } from 'react';

import { ColorContext } from '@ui/context/colors';
import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { edgeIndexer } from './edge-indexer';
import { layoutGraph } from './layout-graph';
import { stackedIndexer, warehouseIndexer } from './node-indexer';
import { stackNodes } from './stack-nodes';

export const reactFlowNodeConstants = {
  CUSTOM_NODE: 'custom-node',
  STACKED_NODE: 'stacked-node'
};

export const useReactFlowPipelineGraph = (
  stages: Stage[],
  warehouses: Warehouse[],
  // basically list of warehouses
  pipeline: string[],
  redraw: boolean,
  stack?: {
    afterNodes?: string[];
  },
  hideSubscriptions?: Record<string, boolean>
) => {
  const { warehouseColorMap } = useContext(ColorContext);

  return useMemo(() => {
    // eslint-disable-next-line prefer-const
    let { graph, stageByName } = layoutGraph(
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
      },
      warehouseColorMap
    );

    const stackedNodes = stackNodes(stack?.afterNodes || [], graph, stageByName);

    graph = stackedNodes.graph;

    layout(graph, { disableOptimalOrderHeuristic: true });

    const reactFlowNodes: Node[] = [];
    const reactFlowEdges: Edge[] = [];

    for (const node of graph.nodes()) {
      const dagreNode = graph.node(node);

      if (stackedIndexer.is(node)) {
        reactFlowNodes.push({
          id: node,
          type: reactFlowNodeConstants.STACKED_NODE,
          position: {
            x: dagreNode?.x - dagreNode?.width / 2,
            y: dagreNode?.y - dagreNode?.height / 2
          },
          data: {
            value: dagreNode?.value,
            id: dagreNode?.id,
            parentNodeId: dagreNode?.parentNodeId
          }
        });
        continue;
      }

      const subscriptionParent = dagreNode?.subscriptionParent?.metadata?.name || '';

      if (hideSubscriptions?.[subscriptionParent]) {
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
      const belongsToWarehouse = warehouseIndexer.getWarehouseName(edge.name || '');

      const dagreEdge = graph.edge(edge);

      reactFlowEdges.push({
        id: edgeIndexer.index(belongsToWarehouse, edge.v, edge.w),
        source: edge.v,
        target: edge.w,
        animated: false,
        type: 'step',
        sourceHandle: belongsToWarehouse,
        targetHandle: belongsToWarehouse,
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: dagreEdge.edgeColor || ''
        },
        style: {
          strokeWidth: 2,
          stroke: dagreEdge.edgeColor || ''
        }
      });
    }

    return {
      nodes: reactFlowNodes,
      edges: reactFlowEdges
    };
  }, [stack?.afterNodes, pipeline, redraw, warehouseColorMap, hideSubscriptions]);
};
