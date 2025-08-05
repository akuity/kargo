import { useMutation, createConnectQueryKey } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { Alert, Modal, message } from 'antd';
import { useParams } from 'react-router-dom';

import { transportWithAuth } from '@ui/config/transport';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getAlias } from '@ui/features/common/utils';
import {
  deleteFreight,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

export const DeleteFreightModal = ({
  visible,
  hide,
  onDelete,
  freight
}: ModalComponentProps & { onDelete: () => void; freight: Freight }) => {
  const { name: project } = useParams();
  const queryClient = useQueryClient();

  const { mutate: deleteAction, isPending } = useMutation(deleteFreight, {
    onSuccess: () => {
      message.success('Freight successfully deleted');

      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          cardinality: 'finite',
          schema: queryFreight,
          input: {
            project
          },
          transport: transportWithAuth
        })
      });
      onDelete();
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
          name: freight.metadata?.name || '',
          project
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
