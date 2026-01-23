import { useEffect } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { RepoSubscription, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { CustomNode } from './custom-node';
import { repoSubscriptionIndexer, stageIndexer, warehouseIndexer } from './node-indexer';
import { DimensionState } from './use-node-dimension-state';

// render nodes to compute the dimension of each node
// that will give us accurate layout for graph
export const DummyNodeRenderrer = (props: {
  stages: Stage[];
  warehouses: WarehouseExpanded[];
  onDimensionChange: (d: DimensionState) => void;
}) => {
  const customNodes: Array<{
    label: string;
    value: WarehouseExpanded | RepoSubscription | Stage;
  }> = [];

  for (const warehouse of props.warehouses) {
    customNodes.push({
      label: warehouseIndexer.index(warehouse),
      value: warehouse
    });

    for (const subscription of warehouse?.spec?.subscriptions || []) {
      customNodes.push({
        label: repoSubscriptionIndexer.index(warehouse, subscription),
        value: subscription
      });
    }
  }

  for (const stage of props.stages) {
    customNodes.push({
      label: stageIndexer.index(stage),
      value: stage
    });
  }

  useEffect(() => {
    const dimensionState: DimensionState = {};
    for (const node of customNodes) {
      const element = document.getElementById(`dummy-${node.label}`);

      if (element) {
        const { width, height } = element.getBoundingClientRect();
        dimensionState[node.label] = { width, height };
      }
    }

    props.onDimensionChange(dimensionState);
  }, [props.onDimensionChange]);

  return (
    <>
      {customNodes.map((node) => (
        <CustomNode id={`dummy-${node.label}`} key={node.label} data={node} />
      ))}
    </>
  );
};
