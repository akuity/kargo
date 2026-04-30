import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import {
  deleteWarehouse,
  listWarehouses
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

export const WarehouseDeleteButton = () => {
  const { name: projectName, warehouseName } = useParams();
  const navigate = useNavigate();
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();

  const { mutate, isPending: isLoadingDelete } = useMutation(deleteWarehouse, {
    onSuccess: () => {
      navigate(generatePath(paths.project, { name: projectName }));

      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: listWarehouses, cardinality: 'finite' })
      });
    }
  });

  const onDelete = () => {
    confirm({
      onOk: () => mutate({ name: warehouseName, project: projectName }),
      title: 'Are you sure you want to delete Warehouse?'
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
