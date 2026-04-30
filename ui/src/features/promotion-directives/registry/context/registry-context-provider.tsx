import { PropsWithChildren } from 'react';

import { useDiscoverPromotionDirectivesRegistries } from '../use-discover-registries';

import { PromotionDirectivesRegistryContext } from './registry-context';

export const PromotionDirectivesRegistryContextProvider = (props: PropsWithChildren) => {
  const registry = useDiscoverPromotionDirectivesRegistries();

  return (
    <PromotionDirectivesRegistryContext.Provider value={{ registry }}>
      {props.children}
    </PromotionDirectivesRegistryContext.Provider>
  );
};
