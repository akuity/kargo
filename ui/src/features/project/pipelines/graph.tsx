import { Controls, ReactFlow } from '@xyflow/react';
import { memo, useEffect } from 'react';

import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { usePipelineContext } from './context/use-pipeline-context';
import { CustomNode } from './nodes/custom-node';
import { FreightTimelineAction } from './types';
import { reactFlowNodeConstants, useReactFlowPipelineGraph } from './utils/use-pipeline-graph';

type GraphProps = {
  project: string;
  stages: Stage[];
  warehouses: Warehouse[];
};

const nodeTypes = {
  [reactFlowNodeConstants.CUSTOM_NODE]: CustomNode
};

export const Graph = memo((props: GraphProps) => {
  const pipelineContext = usePipelineContext();

  const { controlledNodes, controlledEdges } = useReactFlowPipelineGraph(
    props.project,
    props.stages,
    props.warehouses,
    pipelineContext?.hideParents
  );

  useEffect(() => {
    const action = pipelineContext?.state?.action;

    const setEdges = controlledEdges[1];

    if (action === FreightTimelineAction.Promote) {
      setEdges((edges) =>
        edges.map((e) => {
          const isTargetStage = e.target === pipelineContext?.state?.stage;

          const isEdgeBelongsToSelectedWarehouse =
            pipelineContext?.selectedWarehouse === '' ||
            e.targetHandle === pipelineContext?.selectedWarehouse;

          return {
            ...e,
            animated: isTargetStage && isEdgeBelongsToSelectedWarehouse
          };
        })
      );
      return;
    }

    if (action === FreightTimelineAction.PromoteSubscribers) {
      setEdges((edges) =>
        edges.map((e) => {
          return {
            ...e,
            animated: e.source === pipelineContext?.state?.stage
          };
        })
      );
      return;
    }

    setEdges((edges) =>
      edges.map((e) => {
        return {
          ...e,
          animated: false
        };
      })
    );
  }, [controlledEdges[1], pipelineContext?.state, pipelineContext?.selectedWarehouse]);

  return (
    <ReactFlow
      nodes={controlledNodes[0]}
      edges={controlledEdges[0]}
      nodeTypes={nodeTypes}
      fitView
      minZoom={0.1}
      nodesConnectable={false}
      nodesDraggable={false}
      proOptions={{ hideAttribution: true }}
    >
      <Controls />
    </ReactFlow>
  );
});
