import { useEffect, useMemo } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { StackedNodeBody } from '@ui/features/project/pipelines/nodes/stacked-nodes';
import { RepoSubscription, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { CustomNode } from './custom-node';
import { repoSubscriptionIndexer, stageIndexer, warehouseIndexer } from './node-indexer';
import { STACKED_NODE_DUMMY_KEY } from './node-sizer';
import { DimensionState } from './use-node-dimension-state';

// render nodes to compute the dimension of each node
// that will give us accurate layout for graph
export const DummyNodeRenderrer = (props: {
  stages: Stage[];
  warehouses: WarehouseExpanded[];
  knownDimensions: DimensionState;
  onDimensionChange: (d: DimensionState) => void;
}) => {
  const allCustomNodes = useMemo(() => {
    const nodes: Array<{
      label: string;
      value: WarehouseExpanded | RepoSubscription | Stage;
    }> = [];

    for (const warehouse of props.warehouses) {
      nodes.push({
        label: warehouseIndexer.index(warehouse),
        value: warehouse
      });

      for (const subscription of warehouse?.spec?.subscriptions || []) {
        nodes.push({
          label: repoSubscriptionIndexer.index(warehouse, subscription),
          value: subscription
        });
      }
    }

    for (const stage of props.stages) {
      nodes.push({
        label: stageIndexer.index(stage),
        value: stage
      });
    }

    return nodes;
  }, [props.warehouses, props.stages]);

  const unmeasuredNodes = useMemo(
    () => allCustomNodes.filter((n) => !props.knownDimensions[n.label]),
    [allCustomNodes, props.knownDimensions]
  );

  const needsStackedNode = !props.knownDimensions[STACKED_NODE_DUMMY_KEY];

  // Stable string key so the effect only re-runs when the set of unmeasured
  // nodes actually changes, not on every render.
  const unmeasuredKey = useMemo(
    () =>
      unmeasuredNodes
        .map((n) => n.label)
        .sort()
        .join(','),
    [unmeasuredNodes]
  );

  useEffect(() => {
    if (!unmeasuredNodes.length && !needsStackedNode) {
      return;
    }

    const dimensionState: DimensionState = {};

    for (const node of unmeasuredNodes) {
      const element = document.getElementById(`dummy-${node.label}`);

      if (element) {
        const { width, height } = element.getBoundingClientRect();
        dimensionState[node.label] = { width, height };
      }
    }

    if (needsStackedNode) {
      const stackedElement = document.getElementById(`dummy-${STACKED_NODE_DUMMY_KEY}`);
      if (stackedElement) {
        const { width, height } = stackedElement.getBoundingClientRect();
        dimensionState[STACKED_NODE_DUMMY_KEY] = { width, height };
      }
    }

    if (Object.keys(dimensionState).length > 0) {
      props.onDimensionChange(dimensionState);
    }
  }, [unmeasuredKey, needsStackedNode]);

  if (!unmeasuredNodes.length && !needsStackedNode) {
    return null;
  }

  return (
    <>
      {unmeasuredNodes.map((node) => (
        <CustomNode id={`dummy-${node.label}`} key={node.label} data={node} />
      ))}
      {needsStackedNode && <StackedNodeBody id={`dummy-${STACKED_NODE_DUMMY_KEY}`} />}
    </>
  );
};
