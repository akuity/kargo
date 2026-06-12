import { Alert, Modal, message } from 'antd';
import { useParams } from 'react-router-dom';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getAlias } from '@ui/features/common/utils';
import { useDeleteFreight } from '@ui/gen/api/v2/core/core';
import { Freight } from '@ui/gen/api/v2/models';

export const DeleteFreightModal = ({
  visible,
  hide,
  onDelete,
  freight
}: ModalComponentProps & { onDelete: () => void; freight: Freight }) => {
  const { name: project } = useParams();

  const { mutate: deleteAction, isPending } = useDeleteFreight({
    mutation: {
      onSuccess: () => {
        message.success('Freight successfully deleted');
        onDelete();
      }
    }
  });

  const alias = getAlias(freight);

  return (
    <Modal
      destroyOnClose
      open={visible}
      title='Confirm Delete'
      onCancel={hide}
      onOk={() =>
        deleteAction({
          freightNameOrAlias: freight.metadata?.name || '',
          project: project || ''
        })
      }
      okText='Delete'
      okButtonProps={{ loading: isPending, danger: true }}
    >
      <Alert
        type='error'
        banner
        message={
          <div>
            Are you sure you want to delete freight{' '}
            <span className='font-semibold'>{alias ? alias : freight?.metadata?.name}</span>?
          </div>
        }
        className='mb-4'
        showIcon={false}
      />
    </Modal>
  );
};
