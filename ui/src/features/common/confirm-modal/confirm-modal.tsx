import { Modal } from 'antd';

import { ModalProps } from '../modal/use-modal';

export interface ConfirmProps {
  title: string | React.ReactNode;
  onOk: () => void;
  content?: string | React.ReactNode;
}

export const ConfirmModal = ({
  onOk,
  title = 'Are you sure?',
  content,
  hide,
  visible
}: ConfirmProps & ModalProps) => {
  const onConfirm = () => {
    onOk();
    hide();
  };

  return (
    <Modal open={visible} onCancel={hide} okText='Confirm' onOk={onConfirm} title={title}>
      {content}
    </Modal>
  );
};
