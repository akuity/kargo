import { ReactFlow } from '@xyflow/react';

import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

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

  return (
    <ReactFlow
      nodes={graph.nodes}
      edges={graph.edges}
      nodeTypes={nodeTypes}
      fitView
      minZoom={0.1}
      proOptions={{ hideAttribution: true }}
    >
      {/* <Controls /> */}
    </ReactFlow>
  );
};
