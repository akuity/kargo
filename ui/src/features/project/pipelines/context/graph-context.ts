import { createContext, useContext } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';

export type GraphContextType = {
  warehouseByName: Record<string, WarehouseExpanded>;

  // stacking
  stackedNodesParents: string[];
  onStack(parentNode: string): void;
  onUnstack(parentNode: string): void;

  // false during the first render pass so each CustomNode renders a cheap
  // fixed-size placeholder. ReactFlow always mounts every node once -- even
  // with onlyRenderVisibleElements -- so this skips heavy node bodies on init
  // and lets the real components mount only when actually visible.
  ready: boolean;
};

export const GraphContext = createContext<GraphContextType | null>(null);

export const useGraphContext = () => useContext(GraphContext);
