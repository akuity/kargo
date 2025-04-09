import { createContext, useContext } from 'react';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { timerangeTypes } from '../filter-timerange-utils';

export type FreightTimelineControllerContextType = {
  viewingFreight: Freight | null;
  setViewingFreight: (freight: Freight | null) => void;
  preferredFilter: {
    showAlias: boolean;
    artifactCarousel: {
      enabled: boolean;
      state?: {
        repoURL: string;
      };
    };
    sources: string[]; // repoURL
    timerange: timerangeTypes;
  };
  setPreferredFilter: (filter: FreightTimelineControllerContextType['preferredFilter']) => void;
};

export const FreightTimelineControllerContext =
  createContext<FreightTimelineControllerContextType | null>(null);

export const useFreightTimelineControllerContext = () =>
  useContext(FreightTimelineControllerContext);
