import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import React from 'react';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

type Subpage = {
  label: string;
  icon: IconDefinition;
  path: string;
  component: () => React.ReactNode;
};

export type ExtensionStageTabComponentProps = {
  stage: Stage;
};

export type ExtensionStageTab = {
  type: 'stageTab';
  component: ({ stage }: ExtensionStageTabComponentProps) => React.ReactNode;
  label: string;
  icon?: React.ReactNode;
};

export type LayoutExtension = {
  type: 'layoutExtension';
  component: () => React.ReactNode;
};

export type ProjectSubpage = Subpage & {
  type: 'projectSubpage';
};

export type AppSubpage = Subpage & {
  type: 'appSubpage';
};

export type Extension = ExtensionStageTab | LayoutExtension | ProjectSubpage | AppSubpage;
