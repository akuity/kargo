import { Controls, ReactFlow } from '@xyflow/react';
import { useMemo } from 'react';

import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { GraphContext } from '../context/graph-context';

import { CustomNode } from './custom-node';
import { reactFlowNodeConstants, useReactFlowPipelineGraph } from './use-pipeline-graph';

type GraphProps = {
  warehouses: Warehouse[];
  stages: Stage[];
};

const nodeTypes = {
  [reactFlowNodeConstants.CUSTOM_NODE]: CustomNode
};

export const Graph = (props: GraphProps) => {
  const graph = useReactFlowPipelineGraph(props.stages, props.warehouses);

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
        nodes={graph.nodes}
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
