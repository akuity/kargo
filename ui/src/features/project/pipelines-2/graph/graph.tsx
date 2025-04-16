import { Controls, ReactFlow, useNodesState } from '@xyflow/react';
import { useMemo } from 'react';

import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { GraphContext } from '../context/graph-context';

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
  [reactFlowNodeConstants.CUSTOM_NODE]: CustomNode
};

export const Graph = (props: GraphProps) => {
  const graph = useReactFlowPipelineGraph(props.stages, props.warehouses);

  const [nodes, setNodes] = useNodesState(graph.nodes);

  useEventsWatcher(props.project, {
    onStage(stage) {
      const index = stageIndexer.index(stage);
      setNodes((nodes) =>
        nodes.map((node) => {
          if (node.id === index) {
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
          if (node.id === index) {
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
    <GraphContext.Provider value={{ warehouseByName }}>
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
