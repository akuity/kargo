import { Edge, MarkerType, Node, useEdgesState, useNodesState } from '@xyflow/react';
import { graphlib, layout } from 'dagre';
import { useEffect, useMemo } from 'react';

import { ColorMap, getColors } from '@ui/features/stage/utils';
import { RepoSubscription, Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';

import {
  AnyNodeType,
  BoxType,
  ConnectorsType,
  DagreNode,
  NewWarehouseNode,
  NodeType,
  RepoNodeDimensions,
  RepoNodeType,
  getNodeDimensions
} from '../types';

import { getConnectors, initNode, newSubscriptionNode } from './graph';
import { IndexCache } from './index-cache';

const initializeNodes = (
  warehouses: Warehouse[],
  stages: Stage[],
  project?: string
): [AnyNodeType[], ColorMap] => {
  const warehouseNodeMap = {} as { [key: string]: RepoNodeType };
  const nodes = [];

  (warehouses || []).forEach((w: Warehouse) => {
    const warehouseName = w?.metadata?.name;
    if (warehouseName) {
      w?.spec?.subscriptions?.forEach((sub) => {
        nodes.push(newSubscriptionNode(sub, warehouseName));
      });
      warehouseNodeMap[warehouseName] = NewWarehouseNode(w);
    }
  });

  stages.forEach((stage) => {
    (stage.spec?.requestedFreight || []).forEach((f) => {
      if (f?.origin?.kind === 'Warehouse' && f?.sources?.direct) {
        const warehouseName = f.origin?.name;
        if (warehouseName) {
          // the warehouse node will already exist, unless a stage references a missing warehouse
          warehouseNodeMap[warehouseName] = {
            ...warehouseNodeMap[warehouseName],
            stageNames: [
              ...(warehouseNodeMap[warehouseName]?.stageNames || []),
              stage.metadata?.name || ''
            ]
          };
        }
      }
    });

    nodes.push(initNode(stage));
  });

  const warehouseColorMap = getColors(project || '', warehouses, 'warehouses');

  nodes.push(...Object.values(warehouseNodeMap));
  return [nodes, warehouseColorMap];
};

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
  warehouses: Warehouse[]
) => {
  const calculatedNodesAndEdges = useMemo(() => {
    if (!project && stages?.length === 0 && warehouses?.length === 0) {
      return {
        nodes: [],
        edges: []
      };
    }

    const graph = new graphlib.Graph<GraphMeta>({ multigraph: true });

    graph.setGraph({ rankdir: 'LR', ranksep: 100 });
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
        // check if source is warehouse
        if (requestedOrigin?.sources?.direct) {
          const warehouseNodeIndex = `${requestedOrigin.origin?.name}`;

          graph.setEdge(warehouseNodeIndex, stageNodeIndex);
        }

        // check if source is other stage
        for (const sourceStage of requestedOrigin?.sources?.stages || []) {
          graph.setEdge(sourceStage, stageNodeIndex);
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
          x: dagreNode?.x,
          y: dagreNode?.y
        },
        initialWidth: dagreNode?.width,
        initialHeight: dagreNode?.height,
        data: {
          label: node,
          value: dagreNode?.warehouse || dagreNode?.subscription || dagreNode?.stage,
          warehouses: warehouses?.length || 0
        }
      });
    }

    for (const edge of graph.edges()) {
      reactFlowEdges.push({
        id: `${edge.v}->${edge.w}`,
        source: edge.v,
        target: edge.w,
        animated: false,
        markerEnd: {
          type: MarkerType.ArrowClosed
        },
        style: {
          strokeWidth: 2,
          visibility: 'visible'
        }
      });
    }

    return {
      nodes: reactFlowNodes,
      edges: reactFlowEdges
    };
  }, [project, stages, warehouses]);

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

export const usePipelineGraph = (
  project: string | undefined,
  stages: Stage[],
  warehouses: Warehouse[]
): [DagreNode[], ConnectorsType[][], BoxType, Stage[], ColorMap, ColorMap] => {
  return useMemo(() => {
    if (!stages || !warehouses || !project) {
      return [[], [] as ConnectorsType[][], {} as BoxType, [] as Stage[], {}, {}];
    }

    const g = new graphlib.Graph({ multigraph: true });
    g.setGraph({ rankdir: 'LR' });
    g.setDefaultEdgeLabel(() => ({}));

    const [myNodes, warehouseColorMap] = initializeNodes(warehouses, stages, project);
    const parentIndexCache = new IndexCache((node, warehouseName) => {
      return node.type === NodeType.WAREHOUSE && node.warehouseName === warehouseName;
    });
    const subscriberIndexCache = new IndexCache((node, stageName) => {
      return node.type === NodeType.STAGE && node.data.metadata?.name === stageName;
    });

    // add nodes and edges to graph
    myNodes.forEach((item, index) => {
      g.setNode(String(index), getNodeDimensions(item.type));

      if (item.type === NodeType.STAGE) {
        const stage = item.data as Stage;
        const curStageName = stage?.metadata?.name || '';

        (stage?.spec?.requestedFreight || []).forEach((req, i) => {
          if (req.origin?.kind === 'Warehouse') {
            req.sources?.stages?.forEach((upstreamStage) => {
              const to = String(subscriberIndexCache.get(upstreamStage, myNodes));
              const from = String(index);
              g.setEdge(
                to,
                from,
                {
                  color: warehouseColorMap[req?.origin?.name || '']
                },
                `${upstreamStage} ${curStageName} ${i}`
              );
            });
          }
        });
      } else if (item.type === NodeType.WAREHOUSE) {
        // this is a warehouse node
        let i = 0;
        const warehouseName = (item.data as Warehouse).metadata?.name || '';
        for (const stageName of item.stageNames || []) {
          // draw edge between warehouse and stage(s)
          g.setEdge(
            String(index),
            String(subscriberIndexCache.get(stageName, myNodes)),
            {
              color: warehouseColorMap[item.warehouseName]
            },
            `${warehouseName} ${stageName} ${i}`
          );
          i++;
        }
      } else {
        // this is a subscription node
        // draw edge between subscription and warehouse
        g.setEdge(
          String(index),
          String(parentIndexCache.get(item.warehouseName, myNodes)),
          {
            color: warehouseColorMap[item.warehouseName]
          },
          // segregating multiple subscription is important in graph
          // the reason being, we want multiple subscription source to be uniquely identified such that "warehouse" can backtrack accurately
          // without this, graph saw it as if warehouse has single subscription and broke the visual line
          `subscription-${index} ${item.warehouseName} ${index}`
        );
      }
    });

    layout(g, { lablepos: 'c' });

    const nodes = myNodes.map((node, index) => {
      const gNode = g.node(String(index));

      return {
        ...node,
        left: gNode.x - gNode.width / 2,
        top: gNode.y - gNode.height / 2,
        width: gNode.width,
        height: gNode.height
      };
    });

    const connectors = getConnectors(g);

    const box = nodes.reduce(
      (acc, node) => ({
        width: Math.max(acc.width, node.left + node.width),
        height: Math.max(acc.height, node.top + node.height)
      }),
      {
        width: 1000, // creates padding to right of rightmost node
        height: 0
      }
    );

    const sortedStages = nodes
      .filter((item) => item.type === NodeType.STAGE)
      .sort((a, b) => a.left - b.left)
      .map((item) => item.data) as Stage[];

    // color nodes based on stage
    const stageColorMap = getColors(project || '', sortedStages);
    nodes.forEach((node) => {
      if (node.type === NodeType.STAGE) {
        const color = stageColorMap[node.data?.metadata?.name || ''];
        if (color) {
          node.color = color;
        }
      }
    });

    return [nodes, connectors, box, sortedStages, stageColorMap, warehouseColorMap];
  }, [stages, warehouses, project]);
};
