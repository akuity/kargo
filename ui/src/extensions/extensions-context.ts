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
    freightTabs: ctx?.extensions.filter((extension) => extension.type === 'freightTab') || [],
    layoutExtensions:
      ctx?.extensions.filter((extension) => extension.type === 'layoutExtension') || [],
    projectSubpages:
      ctx?.extensions.filter((extension) => extension.type === 'projectSubpage') || [],
    appSubpages: ctx?.extensions.filter((extension) => extension.type === 'appSubpage') || [],
    promoteTabs: ctx?.extensions.filter((extension) => extension.type === 'promoteTab') || [],
    settingsExtensions: ctx?.extensions.filter((extension) => extension.type === 'settings') || [],
    projectSettingsExtensions:
      ctx?.extensions.filter((extension) => extension.type === 'projectSettings') || [],
    projectConfigSubpages:
      ctx?.extensions.filter((extension) => extension.type === 'projectConfigSubpage') || [],
    clusterConfigSubpages:
      ctx?.extensions.filter((extension) => extension.type === 'clusterConfigSubpage') || [],
    argoCDExtension:
      ctx?.extensions.filter((extension) => extension.type === 'argocdExtension')[0] || null,
    promotionStepExtensions:
      ctx?.extensions.filter((extension) => extension.type === 'promotionStep') || []
  };
};
