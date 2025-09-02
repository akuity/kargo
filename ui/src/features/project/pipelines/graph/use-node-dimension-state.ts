import { useState } from 'react';

export type DimensionState = Record<string, { width: number; height: number }>;

export const useNodeDimensionState = () => useState<DimensionState>({});
