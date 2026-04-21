import { useState } from 'react';

export type DimensionState = Record<string, { width: number; height: number }>;

export const useNodeDimensionState = () => {
  const [dimensions, setDimensions] = useState<DimensionState>({});

  return [dimensions, setDimensions] as const;
};
