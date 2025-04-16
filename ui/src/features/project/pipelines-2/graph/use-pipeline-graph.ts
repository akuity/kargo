import { Edge, MarkerType, Node } from '@xyflow/react';
import { graphlib, layout } from 'dagre';
import { useMemo } from 'react';

import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { edgeIndexer } from './edge-indexer';
import { repoSubscriptionIndexer, stageIndexer, warehouseIndexer } from './node-indexer';
import { repoSubscriptionLabelling, stageLabelling, warehouseLabelling } from './node-labeling';
import { repoSubscriptionSizer, stageSizer, warehouseSizer } from './node-sizer';

export const reactFlowNodeConstants = {
  CUSTOM_NODE: 'custom-node'
};

export type GraphMeta = {
  warehouse?: Warehouse;
  subscription?: RepoSubscription;
  stage?: Stage;
  subscriptionParent?: Warehouse;
};

export const useReactFlowPipelineGraph = (stages: Stage[], warehouses: Warehouse[]) => {
  return useMemo(() => {
    const graph = new graphlib.Graph<GraphMeta>({ multigraph: true });

    graph.setGraph({ rankdir: 'LR' });
    graph.setDefaultEdgeLabel(() => ({}));

    const warehouseByName: Record<string, Warehouse> = {};
    const stageByName: Record<string, Stage> = {};

    for (const warehouse of warehouses) {
      warehouseByName[warehouse?.metadata?.name || ''] = warehouse;
    }

    for (const stage of stages) {
      stageByName[stage?.metadata?.name || ''] = stage;
    }

    // subscriptions and warehouses
    for (const warehouse of warehouses) {
      const warehouseNodeIndex = warehouseIndexer.index(warehouse);
      graph.setNode(warehouseNodeIndex, {
        ...warehouseLabelling.label(warehouse),
        ...warehouseSizer.size()
      });

      for (const subscription of warehouse.spec?.subscriptions || []) {
        const subscriptionNodeIndex = repoSubscriptionIndexer.index(warehouse, subscription);

        graph.setNode(subscriptionNodeIndex, {
          ...repoSubscriptionLabelling.label(warehouse, subscription),
          ...repoSubscriptionSizer.size()
        });

        // subscription -> warehouse
        graph.setEdge(subscriptionNodeIndex, warehouseNodeIndex);
      }
    }

    // stages
    for (const stage of stages) {
      const stageNodeIndex = stageIndexer.index(stage);

      graph.setNode(stageNodeIndex, { ...stageLabelling.label(stage), ...stageSizer.size() });

      for (const requestedOrigin of stage.spec?.requestedFreight || []) {
        const warehouseName = requestedOrigin?.origin?.name || '';
        const warehouseNodeIndex = warehouseIndexer.index(warehouseByName[warehouseName]);

        // check if source is warehouse
        if (requestedOrigin?.sources?.direct) {
          graph.setEdge(warehouseNodeIndex, stageNodeIndex);
        }

        // check if source is other stage
        for (const sourceStage of requestedOrigin?.sources?.stages || []) {
          graph.setEdge(
            stageIndexer.index(stageByName[sourceStage]),
            stageNodeIndex,
            {},
            warehouseNodeIndex
          );
        }
      }
    }

    layout(graph, { lablepos: 'c' });

    const reactFlowNodes: Node[] = [];
    const reactFlowEdges: Edge[] = [];

    for (const node of graph.nodes()) {
      const dagreNode = graph.node(node);

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

      reactFlowEdges.push({
        id: edgeIndexer.index(belongsToWarehouse, edge.v, edge.w),
        type: edge.v.startsWith('warehouse') ? 'default' : 'smoothstep',
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
  }, []);
};
