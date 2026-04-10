import { Handle, Position } from '@xyflow/react';
import { PropsWithChildren } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { StageNode } from '../nodes/stage-node';
import { SubscriptionNode } from '../nodes/subscription-node';
import { WarehouseNode } from '../nodes/warehouse-node';

import { repoSubscriptionIndexer, stageIndexer } from './node-indexer';

export const CustomNode = (props: {
  data: {
    label: string;
    value: WarehouseExpanded | RepoSubscription | Stage;
    subscriptionParent?: Warehouse;
    handleOffsetY?: number;
  };
  id?: string;
}) => {
  if (!props.data.value) {
    return null;
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Warehouse') {
    return (
      <CustomNode.Container
        id={props.id}
        warehouse={props.data.value}
        handleOffsetY={props.data.handleOffsetY}
      >
        <WarehouseNode warehouse={props.data.value} />
      </CustomNode.Container>
    );
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription') {
    return (
      <CustomNode.Container
        id={props.id}
        // @ts-expect-error parent is there when value is RepoSubscription, check use-pipeline-graph.ts
        repoSubscription={{ data: props.data.value, parent: props.data.subscriptionParent }}
        handleOffsetY={props.data.handleOffsetY}
      >
        <SubscriptionNode subscription={props.data.value} />
      </CustomNode.Container>
    );
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Stage') {
    return (
      <CustomNode.Container
        id={props.id}
        stage={props.data.value}
        handleOffsetY={props.data.handleOffsetY}
      >
        <StageNode stage={props.data.value} />
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
    // Fixed pixel distance from the node's top edge to the dagre center point.
    // Computed at layout time from the initially measured node height so handles
    // stay anchored at the correct position even when node content changes size.
    handleOffsetY?: number;
  }>
) => {
  let id = '';

  const Children = (
    <div id={props.id} className='w-fit nodrag cursor-default'>
      {props.children}
    </div>
  );

  if (props.stage) {
    id = stageIndexer.index(props.stage);

    const howManyStagesDoThisStageSubscribe = props.stage.spec?.requestedFreight?.length || 0;

    const handleTop = (idx: number) => {
      if (props.handleOffsetY !== undefined) {
        const offset = -((howManyStagesDoThisStageSubscribe - 1) * EDGE_GAP) / 2 + idx * EDGE_GAP;
        return `${props.handleOffsetY + offset}px`;
      }
      return `${50 - ((howManyStagesDoThisStageSubscribe - 1) * EDGE_GAP) / 2 + idx * EDGE_GAP}%`;
    };

    const centerHandleTop = props.handleOffsetY !== undefined ? `${props.handleOffsetY}px` : '50%';

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
              backgroundColor: 'transparent'
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
              backgroundColor: 'transparent'
            }}
          />
        ))}
        <Handle
          type='source'
          position={Position.Right}
          style={{ top: centerHandleTop, backgroundColor: 'transparent' }}
        />
      </>
    );
  }

  if (props.warehouse) {
    id = props.warehouse?.metadata?.name || '';
  }

  if (props.repoSubscription) {
    id = repoSubscriptionIndexer.index(props.repoSubscription.parent, props.repoSubscription.data);
  }

  const singleHandleTop = props.handleOffsetY !== undefined ? `${props.handleOffsetY}px` : '50%';

  return (
    <>
      <Handle
        id={id}
        type='target'
        position={Position.Left}
        style={{
          top: singleHandleTop,
          backgroundColor: 'transparent',
          stroke: 'none',
          border: 'none'
        }}
      />
      {Children}
      <Handle
        id={id}
        type='source'
        position={Position.Right}
        style={{
          top: singleHandleTop,
          backgroundColor: 'transparent',
          stroke: 'none',
          border: 'none'
        }}
      />
    </>
  );
};

const EDGE_GAP = 10;
