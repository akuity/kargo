import { createContext, useContext } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';

export type GraphContextType = {
  warehouseByName: Record<string, WarehouseExpanded>;

  // stacking
  stackedNodesParents: string[];
  onStack(parentNode: string): void;
  onUnstack(parentNode: string): void;
};

export const GraphContext = createContext<GraphContextType | null>(null);

export const useGraphContext = () => useContext(GraphContext);
