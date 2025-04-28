import { Handle, Position } from '@xyflow/react';
import { PropsWithChildren } from 'react';

import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { StageNode } from '../nodes/stage-node';
import { SubscriptionNode } from '../nodes/subscription-node';
import { WarehouseNode } from '../nodes/warehouse-node';

import { repoSubscriptionIndexer, stageIndexer, warehouseIndexer } from './node-indexer';

export const CustomNode = (props: {
  data: {
    label: string;
    value: Warehouse | RepoSubscription | Stage;
    subscriptionParent?: Warehouse;
  };
}) => {
  if (!props.data.value) {
    return null;
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Warehouse') {
    return (
      <CustomNode.Container warehouse={props.data.value}>
        <WarehouseNode warehouse={props.data.value} />
      </CustomNode.Container>
    );
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription') {
    return (
      <CustomNode.Container
        // @ts-expect-error parent is there when value is RepoSubscription, check use-pipeline-graph.ts
        repoSubscription={{ data: props.data.value, parent: props.data.subscriptionParent }}
      >
        <SubscriptionNode subscription={props.data.value} />
      </CustomNode.Container>
    );
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Stage') {
    return (
      <CustomNode.Container stage={props.data.value}>
        <StageNode stage={props.data.value} />
      </CustomNode.Container>
    );
  }

  return <>Unknown Node</>;
};

CustomNode.Container = (
  props: PropsWithChildren<{
    stage?: Stage;
    warehouse?: Warehouse;
    repoSubscription?: { data: RepoSubscription; parent: Warehouse };
  }>
) => {
  let id = '';

  const Children = <div className='max-w-[356px] min-w-[250px]'>{props.children}</div>;

  if (props.stage) {
    id = stageIndexer.index(props.stage);

    const howManyStagesDoThisStageSubscribe = props.stage.spec?.requestedFreight?.length || 0;

    return (
      <>
        {props.stage?.spec?.requestedFreight?.map((freight, idx) => (
          <Handle
            key={idx}
            id={freight?.origin?.name}
            type='target'
            position={Position.Left}
            style={{
              top: `${50 - howManyStagesDoThisStageSubscribe + idx * EDGE_GAP}%`,
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
              top: `${50 - howManyStagesDoThisStageSubscribe + idx * EDGE_GAP}%`,
              backgroundColor: 'transparent'
            }}
          />
        ))}
      </>
    );
  }

  if (props.warehouse) {
    id = warehouseIndexer.index(props.warehouse);
  }

  if (props.repoSubscription) {
    id = repoSubscriptionIndexer.index(props.repoSubscription.parent, props.repoSubscription.data);
  }

  return (
    <>
      <Handle id={id} type='target' position={Position.Left} />
      {Children}
      <Handle id={id} type='source' position={Position.Right} />
    </>
  );
};

const EDGE_GAP = 10;
