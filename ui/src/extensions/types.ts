import React from 'react';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export type ExtensionStageTabComponentProps = {
  stage: Stage;
};

export type ExtensionStageTab = {
  type: 'stageTab';
  component: ({ stage }: ExtensionStageTabComponentProps) => React.ReactNode;
  label: string;
  icon?: React.ReactNode;
};

export type LayoutExtensionTab = {
  type: 'layoutExtension';
  component: () => React.ReactNode;
};

export type Extension = ExtensionStageTab | LayoutExtensionTab;
