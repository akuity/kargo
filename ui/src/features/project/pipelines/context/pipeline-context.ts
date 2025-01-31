import { createContext } from 'react';

import { Freight, Stage } from '@ui/gen/v1alpha1/generated_pb';

import { FreightTimelineAction } from '../types';
import { PipelineStateHook } from '../utils/state';

export interface PipelineContextType {
  state: PipelineStateHook;
  fullFreightById: { [key: string]: Freight };
  subscribersByStage: { [key: string]: Set<string> };
  highlightedStages: { [key: string]: boolean };
  autoPromotionMap: { [key: string]: boolean };
  selectedWarehouse: string;
  setSelectedWarehouse: (newWarehouse: string) => void;
  project: string;
  onHover: (hover: boolean, id: string, isStage?: boolean) => void;
  onPromoteClick: (stage: Stage, type: FreightTimelineAction) => void;
}

export const PipelineContext = createContext<PipelineContextType | null>(null);
