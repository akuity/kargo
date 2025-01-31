import { Handle, Position } from '@xyflow/react';
import { PropsWithChildren } from 'react';

import { RepoSubscription, Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';

import styles from './custom-node.module.less';
import { StageNode } from './stage-node';
import { SubscriptionNode } from './subscription-node';
import { WarehouseNode } from './warehouse-node';

export const CustomNode = ({
  data
}: {
  data: {
    label: string;
    value: Warehouse | RepoSubscription | Stage;
    warehouses?: number;
  };
}) => {
  // todo: why there'd be no data.value?
  if (!data.value) {
    return null;
  }

  if (data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Warehouse') {
    return (
      <CustomNode.Container warehouse={data.value}>
        <WarehouseNode warehouse={data.value} warehouses={data.warehouses} />
      </CustomNode.Container>
    );
  }

  if (data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription') {
    return (
      <CustomNode.Container>
        <SubscriptionNode subscription={data.value} />
      </CustomNode.Container>
    );
  }

  if (data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Stage') {
    return (
      <CustomNode.Container stage={data.value}>
        <StageNode stage={data.value} />
      </CustomNode.Container>
    );
  }

  return <CustomNode.Container>Unknown Node</CustomNode.Container>;
};

CustomNode.Container = (props: PropsWithChildren<{ stage?: Stage; warehouse?: Warehouse }>) => {
  if (props.stage?.metadata?.name) {
    const howManyStagesDoThisStageSubscribe = props.stage?.spec?.requestedFreight?.length || 0;

    return (
      <>
        {props.stage?.spec?.requestedFreight?.map((freight, idx) => (
          <Handle
            key={idx}
            id={freight.origin?.name}
            type='target'
            position={Position.Left}
            style={{ top: `${50 - howManyStagesDoThisStageSubscribe + idx * EDGE_GAP}%` }}
          />
        ))}
        <div className={styles.container}>{props.children}</div>
        {props.stage?.spec?.requestedFreight?.map((freight, idx) => (
          <Handle
            key={idx}
            id={freight.origin?.name}
            type='source'
            position={Position.Right}
            style={{ top: `${50 - howManyStagesDoThisStageSubscribe + idx * EDGE_GAP}%` }}
          />
        ))}
      </>
    );
  }

  return (
    <>
      <Handle id={props.warehouse?.metadata?.name || ''} type='target' position={Position.Left} />
      <div className={styles.container}>{props.children}</div>
      <Handle id={props.warehouse?.metadata?.name || ''} type='source' position={Position.Right} />
    </>
  );
};

const EDGE_GAP = 10;
