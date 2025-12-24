import { graphlib } from '@dagrejs/dagre';

import { ColorMap } from '@ui/features/stage/utils';
import { RepoSubscription, Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

import { repoSubscriptionIndexer, stageIndexer, warehouseIndexer } from './node-indexer';
import { repoSubscriptionLabelling, stageLabelling, warehouseLabelling } from './node-labeling';
import { pickMaxSize, repoSubscriptionSizer, stageSizer, warehouseSizer } from './node-sizer';
import { DimensionState } from './use-node-dimension-state';

export type GraphMeta = {
  warehouse?: Warehouse;
  subscription?: RepoSubscription;
  stage?: Stage;
  subscriptionParent?: Warehouse;
  value?: number;
  id?: string;
  parentNodeId?: string;
};

export const layoutGraph = (
  stage: {
    stages: Stage[];
    ignore?: (s: Stage) => boolean;
  },
  warehouse: {
    warehouses: Warehouse[];
    ignore?: (w: Warehouse) => boolean;
  },
  dimensionState: DimensionState,
  warehouseColorMap?: ColorMap,
  hideSubscriptions?: Record<string, boolean>
) => {
  const graph = new graphlib.Graph<GraphMeta>({ multigraph: true });

  graph.setGraph({ rankdir: 'LR' });
  graph.setDefaultEdgeLabel(() => ({}));

  const warehouseByName: Record<string, Warehouse> = {};
  const stageByName: Record<string, Stage> = {};

  for (const w of warehouse.warehouses) {
    warehouseByName[w?.metadata?.name || ''] = w;
  }

  for (const s of stage.stages) {
    stageByName[s?.metadata?.name || ''] = s;
  }

  for (const w of warehouse.warehouses) {
    if (warehouse.ignore?.(w)) {
      continue;
    }

    const warehouseNodeIndex = warehouseIndexer.index(w);
    graph.setNode(warehouseNodeIndex, {
      ...warehouseLabelling.label(w),
      ...pickMaxSize(warehouseSizer.size(), dimensionState[warehouseNodeIndex] || {})
    });

    if (hideSubscriptions?.[w?.metadata?.name || '']) {
      continue;
    }

    for (const s of w.spec?.subscriptions || []) {
      const subscriptionNodeIndex = repoSubscriptionIndexer.index(w, s);

      graph.setNode(subscriptionNodeIndex, {
        ...repoSubscriptionLabelling.label(w, s),
        ...pickMaxSize(repoSubscriptionSizer.size(), dimensionState[subscriptionNodeIndex] || {})
      });

      graph.setEdge(subscriptionNodeIndex, warehouseNodeIndex);
    }
  }

  for (const s of stage.stages) {
    if (stage.ignore?.(s)) {
      continue;
    }

    const stageNodeIndex = stageIndexer.index(s);

    graph.setNode(stageNodeIndex, {
      ...stageLabelling.label(s),
      ...pickMaxSize(stageSizer.size(), dimensionState[stageNodeIndex] || {})
    });

    for (const requestedOrigin of s.spec?.requestedFreight || []) {
      const warehouseName = requestedOrigin?.origin?.name || '';

      const warehouseData = warehouseByName[warehouseName];

      if (warehouse.ignore?.(warehouseData)) {
        continue;
      }

      const warehouseNodeIndex = warehouseIndexer.index(warehouseByName[warehouseName]);

      const edgeColor = warehouseColorMap?.[warehouseName];

      if (requestedOrigin?.sources?.direct) {
        graph.setEdge(warehouseNodeIndex, stageNodeIndex, { edgeColor }, warehouseNodeIndex);
      }

      for (const sourceStage of requestedOrigin?.sources?.stages || []) {
        graph.setEdge(
          stageIndexer.index(stageByName[sourceStage]),
          stageNodeIndex,
          { edgeColor },
          warehouseNodeIndex
        );
      }
    }
  }

  return { graph, stageByName, warehouseByName };
};
