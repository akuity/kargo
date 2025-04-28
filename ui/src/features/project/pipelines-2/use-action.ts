import { useState } from 'react';

import { ActionContextType, IAction } from './context/action-context';

export const useAction = (): ActionContextType => {
  const [action, setAction] = useState<ActionContextType['action']>();

  return {
    action,
    actPromote(type, stage) {
      setAction({
        type,
        stage
      });
    },
    actManuallyApprove(freight) {
      setAction({
        type: IAction.MANUALLY_APPROVE,
        freight
      });
    },
    cancel: () => setAction(undefined)
  };
};
