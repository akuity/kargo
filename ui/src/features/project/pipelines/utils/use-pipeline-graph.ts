import { graphlib, layout } from 'dagre';
import { useMemo } from 'react';

import { ColorMap, getColors } from '@ui/features/stage/utils';
import { Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';

import {
  AnyNodeType,
  BoxType,
  ConnectorsType,
  DagreNode,
  NewWarehouseNode,
  NodeType,
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
          `subscription ${item.warehouseName} ${index}`
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
