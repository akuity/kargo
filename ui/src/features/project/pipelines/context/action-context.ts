import { createContext, useContext } from 'react';

import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export enum IAction {
  PROMOTE,
  PROMOTE_DOWNSTREAM,
  MANUALLY_APPROVE
}

export type ActionContextType = {
  action?: {
    type: IAction;
    stage?: Stage;
    freight?: Freight;
  };

  actPromote(type: IAction.PROMOTE | IAction.PROMOTE_DOWNSTREAM, stage: Stage): void;
  actManuallyApprove(freight: Freight): void;
  cancel(): void;
};

export const ActionContext = createContext<ActionContextType | null>(null);

export const useActionContext = () => useContext(ActionContext);
