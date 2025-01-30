import { Controls, ReactFlow } from '@xyflow/react';
import { memo, useEffect } from 'react';

import { Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';

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
  const { controlledNodes, controlledEdges } = useReactFlowPipelineGraph(
    props.project,
    props.stages,
    props.warehouses
  );

  const pipelineContext = usePipelineContext();

  useEffect(() => {
    const [edges, setEdges] = controlledEdges;
    const action = pipelineContext?.state?.action;

    let willFallInInfiniteLoopIfSetEdges = false;

    if (action === FreightTimelineAction.Promote) {
      willFallInInfiniteLoopIfSetEdges = Boolean(
        edges?.find((e) => e.target === pipelineContext?.state?.stage && e.animated)
      );
    }

    if (action === FreightTimelineAction.PromoteSubscribers) {
      willFallInInfiniteLoopIfSetEdges = Boolean(
        edges?.find((e) => e.source === pipelineContext?.state?.stage && e.animated)
      );
    }

    if (action === FreightTimelineAction.Promote) {
      if (!willFallInInfiniteLoopIfSetEdges) {
        setEdges(
          edges.map((e) => {
            return {
              ...e,
              animated: e.target === pipelineContext?.state?.stage
            };
          })
        );
      }
      return;
    }

    if (action === FreightTimelineAction.PromoteSubscribers) {
      if (!willFallInInfiniteLoopIfSetEdges) {
        setEdges(
          edges.map((e) => {
            return {
              ...e,
              animated: e.source === pipelineContext?.state?.stage
            };
          })
        );
      }
      return;
    }

    if (edges?.find((e) => e.animated)) {
      setEdges(
        edges.map((e) => {
          return {
            ...e,
            animated: false
          };
        })
      );
    }
  }, [controlledEdges, pipelineContext?.state, pipelineContext?.selectedWarehouse]);

  return (
    <ReactFlow
      nodes={controlledNodes[0]}
      edges={controlledEdges[0]}
      nodeTypes={nodeTypes}
      fitView
      minZoom={0.1}
      nodesConnectable={false}
      nodesDraggable={false}
    >
      <Controls />
    </ReactFlow>
  );
});
