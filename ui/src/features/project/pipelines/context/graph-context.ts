import { createContext, useContext } from 'react';

import { Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

export type GraphContextType = {
  warehouseByName: Record<string, Warehouse>;

  // stacking
  stackedNodesParents: string[];
  onStack(parentNode: string): void;
  onUnstack(parentNode: string): void;
};

export const GraphContext = createContext<GraphContextType | null>(null);

export const useGraphContext = () => useContext(GraphContext);
