import React from 'react';

import { Extension } from './types';

type ExtensionsContextType = {
  extensions: Extension[];
};

export const ExtensionsContext = React.createContext<ExtensionsContextType | null>(null);

export const useExtensionsContext = () => {
  const ctx = React.useContext(ExtensionsContext);

  return {
    stageTabs: ctx?.extensions.filter((extension) => extension.type === 'stageTab') || [],
    layoutExtensions:
      ctx?.extensions.filter((extension) => extension.type === 'layoutExtension') || [],
    projectSubpages:
      ctx?.extensions.filter((extension) => extension.type === 'projectSubpage') || []
  };
};
