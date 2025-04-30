import { ReactFlow } from '@xyflow/react';

import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { nodeTypes } from './mini-graph/constant';
import { useMiniPromotionGraph } from './mini-graph/use-mini-promotion-graph';

type PromotionGraphProps = {
  stage: Stage;
  freight: Freight;
};

export const PromotionGraph = (props: PromotionGraphProps) => {
  const graph = useMiniPromotionGraph(props.stage, props.freight);

  return (
    <div className='bg-zinc-100 w-full h-[300px] rounded-lg'>
      <ReactFlow
        nodeTypes={nodeTypes}
        {...graph}
        fitView
        proOptions={{ hideAttribution: true }}
        panOnDrag={false}
        panOnScroll={false}
        nodesDraggable={false}
        zoomOnScroll={false}
        zoomOnPinch={false}
      />
    </div>
  );
};
