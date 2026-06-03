import { Handle, Position } from '@xyflow/react';
import { Skeleton } from 'antd';

import { RepoSubscription, WarehouseExpanded } from '@ui/extend/types';
import { Stage, Warehouse } from '@ui/gen/api/v2/models';

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
    subscriptionParent?: Warehouse;
    warehouseY?: Record<string, number>;
  };
  id?: string;
}) => {
  const graphContext = useGraphContext();

  const ready = graphContext?.ready || true;

  if (!props.data.value) {
    return null;
  }

  const handleId = props.data?.value?.metadata?.name || '';
  const height = warehouseSizer.size().height;

  const WarehouseNodeBox = ready ? (
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
          left: 2
        }}
      />
      <div id={props.id} className='nodrag cursor-default flex items-center' style={{ height }}>
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
    subscriptionParent: Warehouse;
  };
  id?: string;
}) => {
  const graphContext = useGraphContext();

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

  const StageNodeBox = graphContext?.ready ? (
    <StageNode stage={props.data.value} />
  ) : (
    <NodePlaceholder {...stageSizer.size()} />
  );

  const height = stageSizer.size().height;

  // Sort the per-warehouse handles by the y-coordinate of their source
  // warehouse in the dagre layout. This makes the handles' top-to-bottom
  // order match the warehouses' top-to-bottom order, so edges enter the
  // stage without crossing each other.
  const sortedRequestedFreight = [...(props.data.value.spec?.requestedFreight || [])].sort(
    (a, b) => {
      const yA = props.data.warehouseY?.[a?.origin?.name || ''] ?? 0;
      const yB = props.data.warehouseY?.[b?.origin?.name || ''] ?? 0;
      return yA - yB;
    }
  );

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
            left: 1
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
