import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Typography } from 'antd';

import { queryClient } from '@ui/config/query-client';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import {
  deleteConfigMap,
  listConfigMaps
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

type Props = {
  systemLevel: boolean;
  project: string;
  name: string;
};

export const DeleteConfigMapButton = ({ name, project, systemLevel }: Props) => {
  const confirm = useConfirmModal();

  const { mutate, isPending } = useMutation(deleteConfigMap);

  const onDelete = () => {
    confirm({
      onOk: () => {
        mutate(
          { name, project, systemLevel },
          {
            onSuccess: () =>
              queryClient.refetchQueries({
                queryKey: createConnectQueryKey({
                  schema: listConfigMaps,
                  cardinality: 'finite'
                })
              })
          }
        );
      },
      title: 'Delete ConfigMap',
      content: (
        <>
          Are you sure you want to delete the <Typography.Text strong>{name}</Typography.Text>{' '}
          ConfigMap?
        </>
      )
    });
  };

  return (
    <Button
      variant='filled'
      color='danger'
      size='small'
      icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
      onClick={onDelete}
      loading={isPending}
    >
      Delete
    </Button>
  );
};
