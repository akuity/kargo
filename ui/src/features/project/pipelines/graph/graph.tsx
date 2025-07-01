import { Controls, ReactFlow, useNodesState } from '@xyflow/react';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { queryCache } from '@ui/features/utils/cache';
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

  const stackedNodesParents = filterContext?.preferredFilter?.stackedNodesParents || [];

  const setStackedNodesParents = useCallback(
    (nodes: string[]) =>
      filterContext?.setPreferredFilter({
        ...filterContext?.preferredFilter,
        stackedNodesParents: nodes
      }),
    [filterContext?.setPreferredFilter, filterContext?.preferredFilter]
  );

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

  const [redraw, setRedraw] = useState(false);

  const graph = useReactFlowPipelineGraph(
    props.stages,
    props.warehouses,
    filterContext?.preferredFilter.warehouses || [],
    redraw,
    {
      afterNodes: stackedNodesParents
    },
    filterContext?.preferredFilter?.hideSubscriptions
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

      if (!nodes.find((n) => n.id === index)) {
        setRedraw(!redraw);
      }

      queryCache.imageStageMatrix.update(stage);
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

      if (!nodes.find((n) => n.id === index)) {
        setRedraw(!redraw);
      }

      queryCache.freight.refetch();
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
        proOptions={{ hideAttribution: true }}
      >
        <Controls
          showInteractive={false}
          onFitView={() => {
            setRedraw(!redraw);
          }}
        />
      </ReactFlow>
    </GraphContext.Provider>
  );
};
