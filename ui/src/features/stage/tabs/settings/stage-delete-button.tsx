import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useDeleteStage } from '@ui/gen/api/v2/core/core';

export const StageDeleteButton = () => {
  const { name: projectName, stageName } = useParams();
  const navigate = useNavigate();
  const confirm = useConfirmModal();

  const { mutate, isPending: isLoadingDelete } = useDeleteStage();

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  const onDelete = () => {
    confirm({
      onOk: () => {
        mutate({ stage: stageName || '', project: projectName || '' });
        onClose();
      },
      title: 'Are you sure you want to delete Stage?',
      hide: () => {}
    });
  };

  return (
    <Button
      variant='filled'
      color='danger'
      icon={<FontAwesomeIcon icon={faTrash} size='1x' />}
      onClick={onDelete}
      loading={isLoadingDelete}
    >
      Delete
    </Button>
  );
};
