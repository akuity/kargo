import { faEye, faEyeSlash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Handle, Position } from '@xyflow/react';
import { Button } from 'antd';
import { PropsWithChildren, ReactNode } from 'react';

import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { usePipelineContext } from '../context/use-pipeline-context';

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
  const pipelineContext = usePipelineContext();

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
            style={{
              top: `${50 - howManyStagesDoThisStageSubscribe + idx * EDGE_GAP}%`,
              backgroundColor: 'transparent'
            }}
          />
        ))}
        <div className={styles.container}>{props.children}</div>
        {props.stage?.spec?.requestedFreight?.map((freight, idx) => (
          <Handle
            key={idx}
            id={freight.origin?.name}
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

  let HideParents: ReactNode;
  if (props.warehouse?.metadata?.name) {
    const hideParentsOption = !!props.warehouse;

    const subscription = props.warehouse?.spec?.subscriptions?.[0];

    const repoURL =
      subscription?.image?.repoURL || subscription?.git?.repoURL || subscription?.chart?.repoURL;

    const warehouseNodeIndex = `${props.warehouse?.metadata?.name}-${repoURL}`;

    const parentsHidden = pipelineContext?.hideParents?.includes(warehouseNodeIndex);

    HideParents = hideParentsOption && (
      <Button
        size='small'
        className='absolute top-[50%] translate-y-[-50%] translate-x-[-50%] z-10'
        icon={
          <FontAwesomeIcon
            icon={parentsHidden ? faEye : faEyeSlash}
            onClick={() => {
              const parents = new Set(pipelineContext?.hideParents);

              for (const subscription of props.warehouse?.spec?.subscriptions || []) {
                const repoURL =
                  subscription.image?.repoURL ||
                  subscription.git?.repoURL ||
                  subscription.chart?.repoURL;

                // TODO: centralize node id construction logic
                const nodeIndex = `${props.warehouse?.metadata?.name}-${repoURL}`;

                if (!parentsHidden) {
                  parents.add(nodeIndex);
                } else {
                  parents.delete(nodeIndex);
                }
              }

              pipelineContext?.onHideParents(Array.from(parents));
            }}
          />
        }
      />
    );
  }
  return (
    <>
      {HideParents}
      <Handle
        id={props.warehouse?.metadata?.name || ''}
        type='target'
        position={Position.Left}
        style={{
          backgroundColor: 'transparent',
          top: '48%'
        }}
      />
      <div className={styles.container}>{props.children}</div>
      <Handle
        id={props.warehouse?.metadata?.name || ''}
        type='source'
        position={Position.Right}
        style={{
          backgroundColor: 'transparent',
          top: '48%'
        }}
      />
    </>
  );
};

const EDGE_GAP = 10;
