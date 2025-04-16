import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { StageNode } from '../nodes/stage-node';
import { SubscriptionNode } from '../nodes/subscription-node';
import { WarehouseNode } from '../nodes/warehouse-node';

export const CustomNode = (props: {
  data: {
    label: string;
    value: Warehouse | RepoSubscription | Stage;
  };
}) => {
  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Warehouse') {
    return <WarehouseNode warehouse={props.data.value} />;
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.RepoSubscription') {
    return <SubscriptionNode subscription={props.data.value} />;
  }

  if (props.data.value.$typeName === 'github.com.akuity.kargo.api.v1alpha1.Stage') {
    return <StageNode stage={props.data.value} />;
  }

  return <>Unknown Node</>;
};
