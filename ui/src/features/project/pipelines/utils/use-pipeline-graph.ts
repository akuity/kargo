import { Edge, MarkerType, Node, useEdgesState, useNodesState } from '@xyflow/react';
import { graphlib, layout } from 'dagre';
import { useContext, useEffect, useMemo } from 'react';

import { ColorContext } from '@ui/context/colors';
import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { NodeType, RepoNodeDimensions, getNodeDimensions } from '../types';

export const reactFlowNodeConstants = {
  CUSTOM_NODE: 'custom-node'
};

export type GraphMeta = {
  warehouse?: Warehouse;
  subscription?: RepoSubscription;
  stage?: Stage;
};

export const useReactFlowPipelineGraph = (
  project: string | undefined,
  stages: Stage[],
  warehouses: Warehouse[],
  hiddenParents: string[] = []
) => {
  const { warehouseColorMap } = useContext(ColorContext);

  const calculatedNodesAndEdges = useMemo(() => {
    if (!project && stages?.length === 0 && warehouses?.length === 0) {
      return {
        nodes: [],
        edges: []
      };
    }

    const graph = new graphlib.Graph<GraphMeta>({ multigraph: true });

    graph.setGraph({ rankdir: 'LR', ranksep: 100, edgesep: 0 });
    graph.setDefaultEdgeLabel(() => ({}));

    // subscriptions and warehouses
    for (const warehouse of warehouses) {
      const warehouseNodeIndex = `${warehouse.metadata?.name}`;
      graph.setNode(warehouseNodeIndex, { ...RepoNodeDimensions(), warehouse });

      for (const subscription of warehouse.spec?.subscriptions || []) {
        const repoURL =
          subscription.image?.repoURL || subscription.git?.repoURL || subscription.chart?.repoURL;

        const subscriptionNodeIndex = `${warehouse.metadata?.name}-${repoURL}`;

        graph.setNode(subscriptionNodeIndex, { ...RepoNodeDimensions(), subscription });

        // subscription -> warehouse
        graph.setEdge(subscriptionNodeIndex, warehouseNodeIndex);
      }
    }

    // stages
    for (const stage of stages) {
      const stageNodeIndex = `${stage.metadata?.name}`;
      graph.setNode(stageNodeIndex, { ...getNodeDimensions(NodeType.STAGE), stage });

      for (const requestedOrigin of stage.spec?.requestedFreight || []) {
        const warehouseNodeIndex = `${requestedOrigin.origin?.name}`;

        const edgeColor = warehouseColorMap[warehouseNodeIndex];

        // check if source is warehouse
        if (requestedOrigin?.sources?.direct) {
          graph.setEdge(warehouseNodeIndex, stageNodeIndex, { edgeColor }, warehouseNodeIndex);
        }

        // check if source is other stage
        for (const sourceStage of requestedOrigin?.sources?.stages || []) {
          graph.setEdge(
            sourceStage,
            stageNodeIndex,
            { edgeColor },
            // preserve the source warehouse because there will be multiple edges (for multiple warehouses) for 2 stage
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
        initialWidth: dagreNode?.width,
        initialHeight: dagreNode?.height,
        data: {
          label: node,
          value: dagreNode?.warehouse || dagreNode?.subscription || dagreNode?.stage,
          warehouses: warehouses?.length || 0
        },
        style: {
          visibility: hiddenParents.includes(node) ? 'hidden' : 'visible'
        }
      });
    }

    for (const edge of graph.edges()) {
      const dagreEdge = graph.edge(edge);

      const belongsToWarehouse = edge.name || '';

      reactFlowEdges.push({
        id: `${belongsToWarehouse}-${edge.v}->${edge.w}`,
        type: 'smoothstep',
        source: edge.v,
        target: edge.w,
        animated: false,
        sourceHandle: belongsToWarehouse,
        targetHandle: belongsToWarehouse,
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: dagreEdge?.edgeColor
        },
        style: {
          strokeWidth: 2,
          visibility: hiddenParents.includes(edge.v) ? 'hidden' : 'visible',
          stroke: dagreEdge?.edgeColor
        }
      });
    }

    return {
      nodes: reactFlowNodes,
      edges: reactFlowEdges
    };
  }, [project, stages, warehouses, warehouseColorMap, hiddenParents]);

  const controlledNodes = useNodesState(calculatedNodesAndEdges.nodes);

  const controlledEdges = useEdgesState(calculatedNodesAndEdges.edges);

  useEffect(() => {
    controlledNodes[1](calculatedNodesAndEdges.nodes);
  }, [calculatedNodesAndEdges.nodes]);

  useEffect(() => {
    controlledEdges[1](calculatedNodesAndEdges.edges);
  }, [calculatedNodesAndEdges.edges]);

  return {
    controlledNodes,
    controlledEdges
  };
};
