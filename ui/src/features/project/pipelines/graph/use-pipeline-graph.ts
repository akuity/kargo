import { layout } from '@dagrejs/dagre';
import { Edge, MarkerType, Node } from '@xyflow/react';
import { useContext, useEffect, useRef, useState } from 'react';

import { ColorContext } from '@ui/context/colors';
import { WarehouseExpanded } from '@ui/extend/types';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { edgeIndexer } from './edge-indexer';
import { layoutGraph } from './layout-graph';
import { stackedIndexer, warehouseIndexer } from './node-indexer';
import { STACKED_NODE_DUMMY_KEY, stackSizer } from './node-sizer';
import { stackNodes } from './stack-nodes';
import { DimensionState } from './use-node-dimension-state';

export const reactFlowNodeConstants = {
  CUSTOM_NODE: 'custom-node',
  STACKED_NODE: 'stacked-node'
};

export const useReactFlowPipelineGraph = (
  stages: Stage[],
  warehouses: WarehouseExpanded[],
  // basically list of warehouses
  pipeline: string[],
  redraw: boolean,
  dimensionState: DimensionState,
  stack?: {
    afterNodes?: string[];
  },
  hideSubscriptions?: Record<string, boolean>
) => {
  const { warehouseColorMap } = useContext(ColorContext);

  const [result, setResult] = useState<{ nodes: Node[]; edges: Edge[] }>({
    nodes: [],
    edges: []
  });
  const lastRunRef = useRef(0);
  const functionCalled = useRef(false);

  useEffect(() => {
    if (Object.keys(dimensionState).length === 0) {
      setResult({ nodes: [], edges: [] });
      return;
    }

    const compute = () => {
      lastRunRef.current = Date.now();

      // eslint-disable-next-line prefer-const
      let { graph, stageByName, maxStageHeight } = layoutGraph(
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
        dimensionState,
        warehouseColorMap,
        hideSubscriptions
      );

      graph = stackNodes(stack?.afterNodes || [], graph, stageByName, maxStageHeight);

      layout(graph, { disableOptimalOrderHeuristic: true });

      const reactFlowNodes: Node[] = [];
      const reactFlowEdges: Edge[] = [];

      for (const node of graph.nodes()) {
        const dagreNode = graph.node(node);

        if (stackedIndexer.is(node)) {
          const stackedActualHeight =
            dimensionState[STACKED_NODE_DUMMY_KEY]?.height || stackSizer.size().height;
          reactFlowNodes.push({
            id: node,
            type: reactFlowNodeConstants.STACKED_NODE,
            position: {
              x: dagreNode?.x - dagreNode?.width / 2,
              y: dagreNode?.y - stackedActualHeight / 2
            },
            data: {
              value: dagreNode?.value,
              id: dagreNode?.id,
              parentNodeId: dagreNode?.parentNodeId
            }
          });
          continue;
        }

        // All nodes share a uniform virtual height in dagre (= max stage height)
        // so edges connect at the same center. Use the actual measured height to
        // visually center each node within its virtual slot.
        const actualHeight = dimensionState[node]?.height || dagreNode?.height;

        reactFlowNodes.push({
          id: node,
          type: reactFlowNodeConstants.CUSTOM_NODE,
          position: {
            x: dagreNode?.x - dagreNode?.width / 2,
            y: dagreNode?.y - actualHeight / 2
          },
          data: {
            label: node,
            value: dagreNode?.warehouse || dagreNode?.subscription || dagreNode?.stage,
            subscriptionParent: dagreNode?.subscriptionParent,
            // Fixed pixel offset from the node's top to the dagre center point.
            // Stored at layout time so handles stay anchored even when node content
            // grows and the rendered height changes (node position is not updated).
            handleOffsetY: actualHeight / 2
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
          type:
            (graph.successors(edge.v)?.length || 0) > 1 ||
            (graph.predecessors(edge.w)?.length || 0) > 1
              ? 'step'
              : '',
          sourceHandle: belongsToWarehouse,
          targetHandle: belongsToWarehouse,
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: dagreEdge.edgeColor || ''
          },
          style: {
            strokeWidth: 2,
            stroke: dagreEdge.edgeColor || '',
            transition: 'd 0.3s ease'
          }
        });
      }

      setResult({ nodes: reactFlowNodes, edges: reactFlowEdges });
    };

    if (!functionCalled.current) {
      functionCalled.current = true;
      compute();
      return;
    }

    const elapsed = Date.now() - lastRunRef.current;
    const delay = Math.max(0, 3000 - elapsed);
    const id = setTimeout(compute, delay);
    return () => clearTimeout(id);
  }, [stack?.afterNodes, pipeline, redraw, warehouseColorMap, hideSubscriptions, dimensionState]);

  return result;
};
