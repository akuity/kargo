import { createContext, useContext } from 'react';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export enum IAction {
  PROMOTE,
  PROMOTE_DOWNSTREAM
}

export type ActionContextType = {
  action?: {
    type: IAction;
    stage: Stage;
  };
  act(type: IAction, stage: Stage): void;
  cancel(): void;
};

export const ActionContext = createContext<ActionContextType | null>(null);

export const useActionContext = () => useContext(ActionContext);
