import { createContext } from 'react';

import { PromotionDirectivesRegistry } from '../types';

export interface RegistryContextType {
  registry: PromotionDirectivesRegistry;
}

export const PromotionDirectivesRegistryContext = createContext<RegistryContextType | null>(null);
