import { Controls, ReactFlow, useNodesState } from '@xyflow/react';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';
import { GraphContext } from '../context/graph-context';
import { StackedNodes } from '../nodes/stacked-nodes';

import { CustomNode } from './custom-node';
import { stageIndexer, warehouseIndexer } from './node-indexer';
import { useEventsWatcher } from './use-events-watcher';
import { reactFlowNodeConstants, useReactFlowPipelineGraph } from './use-pipeline-graph';

type GraphProps = {
  warehouses: Warehouse[];
  stages: Stage[];
  project: string;
};

const nodeTypes = {
  [reactFlowNodeConstants.CUSTOM_NODE]: CustomNode,
  [reactFlowNodeConstants.STACKED_NODE]: StackedNodes
};

export const Graph = (props: GraphProps) => {
  const filterContext = useFreightTimelineControllerContext();

  const [stackedNodesParents, setStackedNodesParents] = useState<string[]>([]);

  const onStack = useCallback(
    (parentNode: string) => {
      if (!stackedNodesParents.includes(parentNode)) {
        setStackedNodesParents([...stackedNodesParents, parentNode]);
      }
    },
    [stackedNodesParents]
  );

  const onUnstack = useCallback(
    (parentNode: string) => {
      setStackedNodesParents(stackedNodesParents.filter((node) => node !== parentNode));
    },
    [stackedNodesParents]
  );

  const graph = useReactFlowPipelineGraph(
    props.stages,
    props.warehouses,
    filterContext?.preferredFilter.warehouses || [],
    {
      afterNodes: stackedNodesParents
    }
  );

  const [nodes, setNodes] = useNodesState(graph.nodes);

  useEffect(() => {
    setNodes(graph.nodes);
  }, [graph.nodes]);

  useEventsWatcher(props.project, {
    onStage(stage) {
      const index = stageIndexer.index(stage);
      setNodes((nodes) =>
        nodes.map((node) => {
          if (node.id === index && node.type === reactFlowNodeConstants.CUSTOM_NODE) {
            return {
              ...node,
              data: {
                ...node.data,
                value: stage
              }
            };
          }

          return node;
        })
      );
    },
    onWarehouse(warehouse) {
      const index = warehouseIndexer.index(warehouse);
      setNodes((nodes) =>
        nodes.map((node) => {
          if (node.id === index && node.type === reactFlowNodeConstants.CUSTOM_NODE) {
            return {
              ...node,
              data: {
                ...node.data,
                value: warehouse
              }
            };
          }

          return node;
        })
      );
    }
  });

  const warehouseByName = useMemo(() => {
    const warehouseByName: Record<string, Warehouse> = {};

    for (const warehouse of props.warehouses) {
      warehouseByName[warehouse.metadata?.name || ''] = warehouse;
    }

    return warehouseByName;
  }, [props.warehouses]);

  return (
    <GraphContext.Provider value={{ warehouseByName, stackedNodesParents, onStack, onUnstack }}>
      <ReactFlow
        nodes={nodes}
        edges={graph.edges}
        nodeTypes={nodeTypes}
        fitView
        minZoom={0.1}
        proOptions={{ hideAttribution: true }}
      >
        <Controls />
      </ReactFlow>
    </GraphContext.Provider>
  );
};
