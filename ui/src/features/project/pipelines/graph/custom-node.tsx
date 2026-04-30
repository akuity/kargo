import { Handle, Position } from '@xyflow/react';
import { Skeleton } from 'antd';
import { PropsWithChildren } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { useGraphContext } from '../context/graph-context';
import { StageNode } from '../nodes/stage-node';
import { SubscriptionNode } from '../nodes/subscription-node';
import { WarehouseNode } from '../nodes/warehouse-node';

import { repoSubscriptionIndexer, stageIndexer } from './node-indexer';
import { repoSubscriptionSizer, stageSizer, warehouseSizer } from './node-sizer';

const NodePlaceholder = ({ width, height }: { width: number; height: number }) => (
  <Skeleton.Node active style={{ width, height }} />
);

export const CustomNode = (props: {
  data: {
    label: string;
    value: WarehouseExpanded | RepoSubscription | Stage;
    subscriptionParent?: Warehouse;
  };
  id?: string;
}) => {
  const ready = useGraphContext()?.ready ?? true;

  if (!props.data.value) {
    return null;
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Warehouse') {
    return (
      <CustomNode.Container id={props.id} warehouse={props.data.value}>
        {ready ? (
          <WarehouseNode warehouse={props.data.value} />
        ) : (
          <NodePlaceholder {...warehouseSizer.size()} />
        )}
      </CustomNode.Container>
    );
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription') {
    return (
      <CustomNode.Container
        id={props.id}
        // @ts-expect-error parent is there when value is RepoSubscription, check use-pipeline-graph.ts
        repoSubscription={{ data: props.data.value, parent: props.data.subscriptionParent }}
      >
        {ready ? (
          <SubscriptionNode subscription={props.data.value} />
        ) : (
          <NodePlaceholder {...repoSubscriptionSizer.size()} />
        )}
      </CustomNode.Container>
    );
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Stage') {
    return (
      <CustomNode.Container id={props.id} stage={props.data.value}>
        {ready ? (
          <StageNode stage={props.data.value} />
        ) : (
          <NodePlaceholder {...stageSizer.size()} />
        )}
      </CustomNode.Container>
    );
  }

  return <>Unknown Node</>;
};

CustomNode.Container = (
  props: PropsWithChildren<{
    stage?: Stage;
    warehouse?: WarehouseExpanded;
    repoSubscription?: { data: RepoSubscription; parent: WarehouseExpanded };
    id?: string;
  }>
) => {
  const graphContext = useGraphContext();

  let id = '';
  let height = 0;

  if (props.stage) {
    id = stageIndexer.index(props.stage);
    height = stageSizer.size().height;
  } else if (props.warehouse) {
    id = props.warehouse?.metadata?.name || '';
    height = warehouseSizer.size().height;
  } else if (props.repoSubscription) {
    id = repoSubscriptionIndexer.index(props.repoSubscription.parent, props.repoSubscription.data);
    height = repoSubscriptionSizer.size().height;
  }

  const warehouseHoverProps = props.warehouse
    ? {
        onMouseEnter: () =>
          graphContext?.setHoveredWarehouseName(props.warehouse?.metadata?.name || ''),
        onMouseLeave: () => graphContext?.setHoveredWarehouseName(null)
      }
    : {};

  // Fixed-height slot with content vertically centered. The slot height matches
  // the predefined size used by dagre for layout, so handles -- positioned at
  // 50% of this slot -- line up with the dagre-computed edge endpoints.
  const Children = (
    <div
      id={props.id}
      className='nodrag cursor-default flex items-center'
      style={{ height }}
      {...warehouseHoverProps}
    >
      {props.children}
    </div>
  );

  if (props.stage) {
    const howManyStagesDoThisStageSubscribe = props.stage.spec?.requestedFreight?.length || 0;

    const handleTop = (idx: number) =>
      `calc(50% + ${-((howManyStagesDoThisStageSubscribe - 1) * EDGE_GAP) / 2 + idx * EDGE_GAP}px)`;

    return (
      <>
        {props.stage?.spec?.requestedFreight?.map((freight, idx) => (
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
        {Children}
        {props.stage?.spec?.requestedFreight?.map((freight, idx) => (
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
  }

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
      {Children}
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

const EDGE_GAP = 16;
