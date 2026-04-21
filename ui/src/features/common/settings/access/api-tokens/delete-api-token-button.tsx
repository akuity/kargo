import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { useMutation } from '@tanstack/react-query';
import { Button, Typography } from 'antd';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import {
  deleteProjectAPIToken,
  deleteSystemAPIToken,
  getListProjectAPITokensQueryKey,
  getListSystemAPITokensQueryKey
} from '@ui/gen/api/v2/rbac/rbac';

type Props = {
  systemLevel: boolean;
  project: string;
  name: string;
};

export const DeleteAPITokenButton = ({ name, project, systemLevel }: Props) => {
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();

  const { mutate, isPending: isLoadingDelete } = useMutation({
    mutationFn: () =>
      systemLevel ? deleteSystemAPIToken(name) : deleteProjectAPIToken(project, name),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: systemLevel
          ? getListSystemAPITokensQueryKey()
          : getListProjectAPITokensQueryKey(project)
      })
  });

  const onDelete = () => {
    confirm({
      onOk: () => mutate(),
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
