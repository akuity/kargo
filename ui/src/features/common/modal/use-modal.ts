import React from 'react';

import { ModalComponent, ModalContext } from './modal-context';

export type ModalProps = {
  hide: () => void;
  visible: boolean;
};

const generateModalKey = (() => {
  let count = 0;
  return () => String(++count);
})();

export const useModal = (component?: ModalComponent) => {
  const context = React.useContext(ModalContext);

  if (!context) throw new Error('ModalContextProvider is missing');

  const key = React.useMemo(generateModalKey, []);

  const hide = React.useCallback(() => context.hide(key), [context, key]);
  const show = React.useCallback(
    (newComponent?: ModalComponent) => {
      if (newComponent) {
        context.show(key, newComponent);
      } else if (component) {
        context.show(key, component);
      }
    },
    [key, component, context]
  );

  React.useEffect(() => () => hide(), [hide]);

  return { show, hide };
};
