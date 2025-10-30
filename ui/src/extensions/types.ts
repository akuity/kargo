import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { IconDefinition } from '@fortawesome/free-solid-svg-icons';
import React from 'react';

import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

type Subpage = {
  label: string;
  icon: IconDefinition;
  path: string;
  component: () => React.ReactNode;
};

export type StageTabComponentProps = {
  stage: Stage;
};

export type StageTab = {
  type: 'stageTab';
  component: ({ stage }: StageTabComponentProps) => React.ReactNode;
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

export type PromoteTabComponentProps = {
  stage: Stage;
  freight: Freight;
};

export type PromoteTab = {
  type: 'promoteTab';
  component: (props: PromoteTabComponentProps) => React.ReactNode;
  label: string;
  icon?: React.ReactNode;
};

export type SettingsExtension = {
  type: 'settings';
  component: () => React.ReactNode;
  label: string;
  icon: IconProp;
  path: string;
};

export type ProjectSettingsExtension = {
  type: 'projectSettings';
  component: () => React.ReactNode;
  label: string;
  icon: IconProp;
  path: string;
};

export type ArgoCDExtension = {
  type: 'argocdExtension';
  component: () => React.ReactNode;
};

export type Extension =
  | StageTab
  | LayoutExtension
  | ProjectSubpage
  | AppSubpage
  | PromoteTab
  | SettingsExtension
  | ProjectSettingsExtension
  | ArgoCDExtension;
