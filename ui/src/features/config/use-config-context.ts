import { useContext } from 'react';

import { ConfigContext } from './config-context';

export const useConfigContext = () => {
  const ctx = useContext(ConfigContext);

  if (ctx === null) {
    throw new Error(`useConfigContext must be used within a Provider`);
  }

  return ctx;
};
