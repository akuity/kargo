import { useContext } from 'react';

import { PromotionDirectivesRegistryContext } from './registry-context';

export const usePromotionDirectivesRegistryContext = () => {
  const ctx = useContext(PromotionDirectivesRegistryContext);

  if (ctx === null) {
    throw new Error(`${usePromotionDirectivesRegistryContext.name} must be used within a Provider`);
  }

  return ctx;
};
