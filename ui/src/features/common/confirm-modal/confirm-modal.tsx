import { Modal, ModalFuncProps } from 'antd';
import React from 'react';

export interface ConfirmProps {
  title: string | React.ReactNode;
  onOk: () => Promise<unknown> | void;
  hide?: () => void;
  visible?: boolean;
  content?: string | React.ReactNode;
}

export const ConfirmModal = ({
  onOk,
  onCancel,
  title = 'Are you sure?',
  content,
  hide,
  visible,
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
      open={visible}
      onCancel={cancel}
      okText='Confirm'
      onOk={confirm}
      title={title}
      {...props}
      okButtonProps={{ ...props.okButtonProps, loading }}
    >
      {content}
    </Modal>
  );
};
