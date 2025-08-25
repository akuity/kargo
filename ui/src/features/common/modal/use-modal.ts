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

  // Always initialize hooks in the same order to satisfy the Rules of Hooks
  const key = React.useMemo(generateModalKey, []);

  const hide = React.useCallback(() => {
    if (!context) {
      if (process.env.NODE_ENV === 'development') {
        // eslint-disable-next-line no-console
        console.warn('useModal: ModalContext missing. hide() is a no-op.');
      }
      return;
    }
    context.hide(key);
  }, [context, key]);

  const show = React.useCallback(
    (newComponent?: ModalComponent) => {
      if (!context) {
        if (process.env.NODE_ENV === 'development') {
          // eslint-disable-next-line no-console
          console.warn('useModal: ModalContext missing. show() is a no-op.');
        }
        return;
      }
      if (newComponent) {
        context.show(key, newComponent);
      } else if (component) {
        context.show(key, component);
      }
    },
    [context, key, component]
  );

  React.useEffect(() => () => hide(), [hide]);

  return { show, hide };
};
