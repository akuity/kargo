import { ModalFuncProps } from 'antd';
import React from 'react';

import { useModal } from '../modal/use-modal';

import { ConfirmModal, ConfirmProps } from './confirm-modal';

export const useConfirmModal = () => {
  const { show } = useModal();

  return React.useCallback(
    (propsConfirmModal: ConfirmProps & ModalFuncProps) => {
      show((p) => <ConfirmModal {...propsConfirmModal} {...p} />);
    },
    [show]
  );
};
