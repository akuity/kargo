import { Modal, ModalFuncProps } from 'antd';

export interface ConfirmProps {
  title: string | React.ReactNode;
  onOk: () => void;
  hide?: () => void;
  content?: string | React.ReactNode;
}

export const ConfirmModal = ({
  onOk,
  title = 'Are you sure?',
  content,
  hide,
  visible,
  ...props
}: ConfirmProps & ModalFuncProps) => {
  const onConfirm = () => {
    onOk();
    hide?.();
  };

  return (
    <Modal
      open={visible}
      onCancel={hide}
      okText='Confirm'
      onOk={onConfirm}
      title={title}
      {...props}
    >
      {content}
    </Modal>
  );
};
