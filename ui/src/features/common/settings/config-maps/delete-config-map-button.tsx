import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMutation } from '@tanstack/react-query';
import { Button, Typography } from 'antd';

import { queryClient } from '@ui/config/query-client';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import {
  deleteProjectConfigMap,
  deleteSharedConfigMap,
  getListProjectConfigMapsQueryKey,
  getListSharedConfigMapsQueryKey
} from '@ui/gen/api/v2/core/core';

type Props = {
  project: string;
  name: string;
};

export const DeleteConfigMapButton = ({ name, project }: Props) => {
  const confirm = useConfirmModal();

  const mutationFn = project
    ? () => deleteProjectConfigMap(project, name)
    : () => deleteSharedConfigMap(name);
  const queryKey = project
    ? getListProjectConfigMapsQueryKey(project)
    : getListSharedConfigMapsQueryKey();

  const { mutate, isPending } = useMutation({
    mutationFn,
    onSuccess: () => queryClient.refetchQueries({ queryKey })
  });

  const onDelete = () => {
    confirm({
      onOk: () => mutate(),
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
