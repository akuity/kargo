import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Typography } from 'antd';

import { queryClient } from '@ui/config/query-client';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import {
  deleteAPIToken,
  listAPITokens
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

type Props = {
  systemLevel: boolean;
  project: string;
  name: string;
};

export const DeleteAPITokenButton = ({ name, project, systemLevel }: Props) => {
  const confirm = useConfirmModal();

  const { mutate, isPending: isLoadingDelete } = useMutation(deleteAPIToken);

  const onDelete = () => {
    confirm({
      onOk: () => {
        mutate(
          { name, project, systemLevel },
          {
            onSuccess: () =>
              queryClient.refetchQueries({
                queryKey: createConnectQueryKey({
                  schema: listAPITokens,
                  cardinality: 'finite'
                })
              })
          }
        );
      },
      title: 'Delete API Token',
      content: (
        <>
          Are you sure you want to delete the <Typography.Text strong>{name}</Typography.Text> API
          token? This action cannot be undone, and any services using this token will stop working.
        </>
      )
    });
  };

  return (
    <Button
      variant='filled'
      color='danger'
      size='small'
      icon={<FontAwesomeIcon icon={faTrash} size='1x' />}
      onClick={onDelete}
      loading={isLoadingDelete}
    >
      Delete
    </Button>
  );
};
