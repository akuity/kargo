import { theme } from 'antd';
import { useMemo } from 'react';

import { TargetState, stateOrder, targetStatePresentations } from '../model/target-state';

// useStateColors resolves every state's theme token once, so each bar, dot and
// square on the page paints from one vocabulary and follows dark mode.
export const useStateColors = (): Record<TargetState, string> => {
  const { token } = theme.useToken();
  return useMemo(
    () =>
      Object.fromEntries(
        stateOrder.map((state) => [state, token[targetStatePresentations[state].colorToken]])
      ) as Record<TargetState, string>,
    [token]
  );
};
