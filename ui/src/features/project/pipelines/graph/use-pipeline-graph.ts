import { layout } from '@dagrejs/dagre';
import { Edge, MarkerType, Node } from '@xyflow/react';
import { useContext, useEffect, useRef, useState } from 'react';

import { ColorContext } from '@ui/context/colors';
import { WarehouseExpanded } from '@ui/extend/types';
import { Stage } from '@ui/gen/api/v2/models';

import { edgeIndexer } from './edge-indexer';
import { layoutGraph } from './layout-graph';
import { stackedIndexer, warehouseIndexer } from './node-indexer';
import { repoSubscriptionSizer, stackSizer, stageSizer, warehouseSizer } from './node-sizer';
import { stackNodes } from './stack-nodes';

export const reactFlowNodeConstants = {
  STACKED_NODE: 'stacked-node',
  CUSTOM_WAREHOUSE_NODE: 'custom-warehouse-node',
  CUSTOM_REPO_SUBSCRIPTION_NODE: 'custom-repo-subscription-node',
  CUSTOM_STAGE_NODE: 'custom-stage-node'
};

export const useReactFlowPipelineGraph = (
  stages: Stage[],
  warehouses: WarehouseExpanded[],
  // basically list of warehouses
  pipeline: string[],
  redraw: boolean,
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
        warehouseColorMap,
        hideSubscriptions
      );

      graph = stackNodes(stack?.afterNodes || [], graph, stageByName, maxStageHeight);

      layout(graph, { disableOptimalOrderHeuristic: true });

      const reactFlowNodes: Node[] = [];
      const reactFlowEdges: Edge[] = [];

      // y-coordinate of each warehouse after layout. Stage nodes use this to
      // sort their per-warehouse handles top-to-bottom so edges enter in the
      // same vertical order as the source warehouses, avoiding crossings.
      const warehouseY: Record<string, number> = {};
      for (const node of graph.nodes()) {
        const dagreNode = graph.node(node);
        if (dagreNode?.warehouse) {
          warehouseY[dagreNode.warehouse?.metadata?.name || ''] = dagreNode.y;
        }
      }

      for (const node of graph.nodes()) {
        const dagreNode = graph.node(node);

        if (stackedIndexer.is(node)) {
          const stackedActualHeight = stackSizer.size().height;
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

        let actualHeight: number;
        let customGraphType: string;
        if (dagreNode?.warehouse) {
          customGraphType = reactFlowNodeConstants.CUSTOM_WAREHOUSE_NODE;
          actualHeight = warehouseSizer.size().height;
        } else if (dagreNode?.subscription) {
          customGraphType = reactFlowNodeConstants.CUSTOM_REPO_SUBSCRIPTION_NODE;
          actualHeight = repoSubscriptionSizer.size().height;
        } else {
          customGraphType = reactFlowNodeConstants.CUSTOM_STAGE_NODE;
          actualHeight = stageSizer.size().height;
        }

        reactFlowNodes.push({
          id: node,
          type: customGraphType,
          position: {
            x: dagreNode?.x - dagreNode?.width / 2,
            y: dagreNode?.y - actualHeight / 2
          },
          data: {
            label: node,
            value: dagreNode?.warehouse || dagreNode?.subscription || dagreNode?.stage,
            subscriptionParent: dagreNode?.subscriptionParent,
            warehouseY
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
          type: 'default',
          sourceHandle: belongsToWarehouse,
          targetHandle: belongsToWarehouse,
          data: { warehouseName: belongsToWarehouse },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: '#777',
            width: 6,
            height: 6,
            strokeWidth: 2
          },
          style: {
            strokeWidth: 4,
            stroke: dagreEdge.edgeColor || '#9ca3af',
            strokeOpacity: 0.3,
            transition: 'd 0.3s ease, stroke-opacity 0.5s ease, filter 0.2s ease'
          }
        });
      }

      setResult({ nodes: reactFlowNodes, edges: reactFlowEdges });
    };

    // Run the first layout immediately so the graph paints without delay.
    // Subsequent recomputes are throttled to at most once per 3s, since layout
    // is expensive and watch-driven redraws can fire frequently. The watch
    // hooks coalesce their event bursts (see watch-utils debounce), which keeps
    // this throttle from being starved by a constant stream of redraw triggers.
    if (!functionCalled.current) {
      functionCalled.current = true;
      compute();
      return;
    }

    const elapsed = Date.now() - lastRunRef.current;
    const delay = Math.max(0, 3000 - elapsed);
    const id = setTimeout(compute, delay);
    return () => clearTimeout(id);
  }, [stack?.afterNodes, pipeline, redraw, warehouseColorMap, hideSubscriptions]);

  return result;
};
