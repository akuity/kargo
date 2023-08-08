import { Modal } from 'antd';

import { ModalProps } from '../modal/use-modal';

export interface ConfirmProps {
  title: string;
  onOk: () => void;
}

export const ConfirmModal = ({
  onOk,
  title = 'Are you sure?',
  hide,
  visible
}: ConfirmProps & ModalProps) => {
  const onConfirm = () => {
    onOk();
    hide();
  };

  return (
    <Modal
      closable={false}
      open={visible}
      onCancel={hide}
      okText='Confirm'
      onOk={onConfirm}
      title={title}
    />
  );
};
