import { createContext, useContext } from 'react';

export type Filter = {
  stage: string;
  phase: string[];
  health: string[];
  version: {
    source?: string[];
    version?: string[];
  };
  lastPromotion?: [Date, Date];
};

export type FilterContextType = {
  filters: Filter;
  onFilter: (next: Filter) => void;
};

export const FilterContext = createContext<FilterContextType | null>(null);

export const useFilterContext = () => useContext(FilterContext);
