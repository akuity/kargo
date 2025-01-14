import { createContext } from 'react';

import { Freight } from '@ui/gen/v1alpha1/generated_pb';

import { PipelineStateHook } from '../utils/state';

export interface PipelineContextType {
  state: PipelineStateHook;
  fullFreightById: { [key: string]: Freight };
  subscribersByStage: { [key: string]: Set<string> };
  highlightedStages: { [key: string]: boolean };
  autoPromotionMap: { [key: string]: boolean };
  selectedWarehouse: string;
  project: string;
}

export const PipelineContext = createContext<PipelineContextType | null>(null);
