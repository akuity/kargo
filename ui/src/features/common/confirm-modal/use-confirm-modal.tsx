import React from 'react';

import { useModal } from '../modal/use-modal';

import { ConfirmModal, ConfirmProps } from './confirm-modal';

export const useConfirmModal = () => {
  const { show } = useModal();

  return React.useCallback(
    (propsConfirmModal: ConfirmProps) => {
      show((p) => <ConfirmModal {...propsConfirmModal} {...p} />);
    },
    [show]
  );
};
