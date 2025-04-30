import { createContext, useContext } from 'react';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { timerangeTypes } from '../freight/filter-timerange-utils';

export type FreightTimelineControllerContextType = {
  viewingFreight: Freight | null;
  setViewingFreight: (freight: Freight | null) => void;
  preferredFilter: {
    showColors: boolean;
    showAlias: boolean;
    sources: string[]; // repoURL
    timerange: timerangeTypes;
    warehouses: string[];
    hideUnusedFreights: boolean;
    stackedNodesParents: string[];
    hideSubscriptions: Record<string, boolean>;
    images: boolean;
  };
  setPreferredFilter: (filter: FreightTimelineControllerContextType['preferredFilter']) => void;
};

export const FreightTimelineControllerContext =
  createContext<FreightTimelineControllerContextType | null>(null);

export const useFreightTimelineControllerContext = () =>
  useContext(FreightTimelineControllerContext);
