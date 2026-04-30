import React from 'react';

import { delay } from '@ui/utils/delay';

import { Portal } from '../portal';

import { ModalComponent, ModalContext, ModalItem } from './modal-context';

interface ModalProviderProps {
  children: React.ReactNode;
  container: HTMLElement | null;
}

export const ModalContextProvider = ({ children, container }: ModalProviderProps) => {
  const [modals, setModals] = React.useState<Record<string, ModalItem>>({});

  const show = (key: string, modal: ModalComponent) => {
    setModals((_modals) => ({
      ..._modals,
      [key]: {
        component: modal,
        visible: true
      }
    }));
  };

  const hide = async (key: string) => {
    setModals((_modals) => {
      if (!_modals[key]) {
        return _modals;
      }

      return {
        ..._modals,
        [key]: {
          ..._modals[key],
          visible: false
        }
      };
    });

    // Delay for animation
    await delay(200);

    setModals((_modals) => {
      if (!_modals[key]) {
        return _modals;
      }
      const newModals = { ..._modals };
      delete newModals[key];
      return newModals;
    });
  };

  const contextValue = React.useMemo(() => ({ show, hide }), []);

  return (
    <ModalContext.Provider value={contextValue}>
      {/* REACT SPECIFIC BUG - https://github.com/remix-run/react-router/issues/8834#issuecomment-1118083034 */}
      <div className='h-full'>
        {children}
        <Portal container={container}>
          <>
            {Object.keys(modals).map((key) => {
              const { component: Component, visible } = modals[key];
              return <Component key={key} hide={() => hide(key)} visible={visible} />;
            })}
          </>
        </Portal>
      </div>
    </ModalContext.Provider>
  );
};
