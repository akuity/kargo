import { createContext, useContext } from 'react';

import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export type DictionaryContextType = {
  freightInStages: Record<string, Stage[]>;
  freightById: Record<string, Freight>;
  stageAutoPromotionMap: Record<string, boolean>;
};

export const DictionaryContext = createContext<DictionaryContextType | null>(null);

export const useDictionaryContext = () => useContext(DictionaryContext);
