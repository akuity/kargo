import { createContext, useContext } from 'react';

type RjsfConfig = {
  showDescription: boolean;
};

type RjsfConfigContextType = RjsfConfig & {
  setConfig(nextConfig: RjsfConfig): void;
};

export const RjsfConfigContext = createContext<RjsfConfigContextType>({
  showDescription: false,
  setConfig() {
    // noop
  }
});

export const useRjsfConfigContext = () => useContext(RjsfConfigContext);
