import { Handle, Position } from '@xyflow/react';
import { Skeleton } from 'antd';

import { RepoSubscription, WarehouseExpanded } from '@ui/extend/types';
import { Stage } from '@ui/gen/api/v2/models';

import { useGraphContext } from '../context/graph-context';
import { StageNode } from '../nodes/stage-node';
import { SubscriptionNode } from '../nodes/subscription-node';
import { WarehouseNode } from '../nodes/warehouse-node';

import { repoSubscriptionIndexer } from './node-indexer';
import { repoSubscriptionSizer, stageSizer, warehouseSizer } from './node-sizer';

const NodePlaceholder = ({ width, height }: { width: number; height: number }) => (
  <Skeleton.Node active style={{ width, height }} />
);

export const CustomWarehouseNode = (props: {
  data: {
    label: string;
    value: WarehouseExpanded;
    subscriptionParent?: WarehouseExpanded;
    warehouseY?: Record<string, number>;
  };
  id?: string;
}) => {
  const graphContext = useGraphContext();

  if (!props.data.value) {
    return null;
  }

  const handleId = props.data?.value?.metadata?.name || '';
  const height = warehouseSizer.size().height;

  const WarehouseNodeBox = graphContext?.ready ? (
    <WarehouseNode warehouse={props.data.value} />
  ) : (
    <NodePlaceholder {...warehouseSizer.size()} />
  );

  return (
    <>
      <Handle
        id={handleId}
        type='target'
        position={Position.Left}
        style={{
          top: '50%',
          backgroundColor: 'transparent',
          stroke: 'none',
          border: 'none',
          left: 3
        }}
      />
      <div
        id={props.id}
        className='nodrag cursor-default flex items-center'
        style={{ height }}
        onMouseEnter={() =>
          graphContext?.setHoveredWarehouseName(props.data.value?.metadata?.name || '')
        }
        onMouseLeave={() => graphContext?.setHoveredWarehouseName(null)}
      >
        {WarehouseNodeBox}
      </div>
      <Handle
        id={handleId}
        type='source'
        position={Position.Right}
        style={{
          top: '50%',
          backgroundColor: 'transparent',
          stroke: 'none',
          border: 'none',
          right: 4
        }}
      />
    </>
  );
};

export const CustomRepoSubscriptionNode = (props: {
  data: {
    label: string;
    value: RepoSubscription;
    subscriptionParent: WarehouseExpanded;
  };
  id?: string;
}) => {
  const graphContext = useGraphContext();

  if (!props.data.value) {
    return null;
  }

  const RepoSubscriptionNodeBox = graphContext?.ready ? (
    <SubscriptionNode subscription={props.data.value} />
  ) : (
    <NodePlaceholder {...repoSubscriptionSizer.size()} />
  );

  const id = repoSubscriptionIndexer.index(props.data.subscriptionParent, props.data.value);
  const height = repoSubscriptionSizer.size().height;

  return (
    <>
      <Handle
        id={id}
        type='target'
        position={Position.Left}
        style={{
          top: '50%',
          backgroundColor: 'transparent',
          stroke: 'none',
          border: 'none',
          left: 2
        }}
      />
      <div id={props.id} className='nodrag cursor-default flex items-center' style={{ height }}>
        {RepoSubscriptionNodeBox}
      </div>
      <Handle
        id={id}
        type='source'
        position={Position.Right}
        style={{
          top: '50%',
          backgroundColor: 'transparent',
          stroke: 'none',
          border: 'none',
          right: 4
        }}
      />
    </>
  );
};

export const CustomStageNode = (props: {
  data: {
    label: string;
    value: Stage;
    warehouseY?: Record<string, number>;
  };
  id?: string;
}) => {
  const graphContext = useGraphContext();

  if (!props.data.value) {
    return null;
  }

  const StageNodeBox = graphContext?.ready ? (
    <StageNode stage={props.data.value} />
  ) : (
    <NodePlaceholder {...stageSizer.size()} />
  );

  const height = stageSizer.size().height;

  const warehouseY = props.data.warehouseY;
  const requestedFreight = props.data.value.spec?.requestedFreight || [];

  // Filtering a warehouse out of the pipeline takes its edges with it, so the
  // handles have to be spread over the warehouses still drawn. Spreading them
  // over every requested warehouse would leave the survivors holding the slots
  // they had in the full set, and a lone remaining edge would meet the stage
  // well below its middle. warehouseY is keyed by exactly the warehouses the
  // layout kept, which makes it the record of what is still drawn.
  const drawnFreight = warehouseY
    ? requestedFreight.filter((f) => warehouseY[f?.origin?.name || ''] !== undefined)
    : requestedFreight;

  // A stage whose warehouses are all filtered out is dropped from the layout
  // altogether, so an empty result here means the warehouses went missing some
  // other way. Fall back to the full set rather than strand the edges.
  const visibleFreight = drawnFreight.length ? drawnFreight : requestedFreight;

  // Sort the per-warehouse handles by the y-coordinate of their source
  // warehouse in the dagre layout. This makes the handles' top-to-bottom
  // order match the warehouses' top-to-bottom order, so edges enter the
  // stage without crossing each other.
  const sortedRequestedFreight = [...visibleFreight].sort((a, b) => {
    const yA = warehouseY?.[a?.origin?.name || ''] ?? 0;
    const yB = warehouseY?.[b?.origin?.name || ''] ?? 0;
    return yA - yB;
  });

  const handleTop = (idx: number) =>
    `calc(50% + ${-((sortedRequestedFreight.length - 1) * EDGE_GAP) / 2 + idx * EDGE_GAP}px)`;

  return (
    <>
      {sortedRequestedFreight.map((freight, idx) => (
        <Handle
          key={idx}
          id={freight?.origin?.name}
          type='target'
          position={Position.Left}
          style={{
            top: handleTop(idx),
            backgroundColor: 'transparent',
            border: 'none',
            left: 2
          }}
        />
      ))}
      <div id={props.id} className='nodrag cursor-default flex items-center' style={{ height }}>
        {StageNodeBox}
      </div>
      {sortedRequestedFreight.map((freight, idx) => (
        <Handle
          key={idx}
          id={freight?.origin?.name}
          type='source'
          position={Position.Right}
          style={{
            top: handleTop(idx),
            backgroundColor: 'transparent',
            border: 'none',
            right: 4
          }}
        />
      ))}
      <Handle
        type='source'
        position={Position.Right}
        style={{ top: '50%', backgroundColor: 'transparent', border: 'none', right: 4 }}
      />
    </>
  );
};

const EDGE_GAP = 16;
