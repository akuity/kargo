import { useEffect, useState } from 'react';

import { ActionContextType, IAction } from './context/action-context';
import { escapeCancelsAction } from './escape-cancels-action';

export const useAction = (): ActionContextType => {
  const [action, setAction] = useState<ActionContextType['action']>();

  const active = !!action;

  useEffect(() => {
    if (!active) {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (escapeCancelsAction(event, document)) {
        setAction(undefined);
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [active]);

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
