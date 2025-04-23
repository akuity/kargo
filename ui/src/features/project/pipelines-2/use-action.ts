import { useState } from 'react';

import { ActionContextType } from './context/action-context';

export const useAction = (): ActionContextType => {
  const [action, setAction] = useState<ActionContextType['action']>();

  return {
    action,
    act: (type, stage) => setAction({ type, stage }),
    cancel: () => setAction(undefined)
  };
};
