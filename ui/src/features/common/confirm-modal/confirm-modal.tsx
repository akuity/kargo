import { Modal, ModalFuncProps } from 'antd';
import React from 'react';

export interface ConfirmProps {
  title: string | React.ReactNode;
  onOk: () => Promise<unknown> | void;
  hide?: () => void;
  content?: string | React.ReactNode;
}

export const ConfirmModal = ({
  onOk,
  onCancel,
  title = 'Are you sure?',
  content,
  hide,
  ...props
}: ConfirmProps & ModalFuncProps) => {
  const [loading, setLoading] = React.useState(false);

  const confirm = async () => {
    try {
      setLoading(true);
      await onOk();
      hide?.();
    } finally {
      setLoading(false);
    }
  };

  const cancel = () => {
    if (loading) return;

    onCancel?.();
    hide?.();
  };

  return (
    <Modal
      onCancel={cancel}
      okText='Confirm'
      onOk={confirm}
      title={title}
      okButtonProps={{ loading }}
      {...props}
    >
      {content}
    </Modal>
  );
};
