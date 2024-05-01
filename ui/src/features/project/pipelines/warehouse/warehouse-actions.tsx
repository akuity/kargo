import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import { faPen, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Space } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import {
  deleteWarehouse,
  listWarehouses
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Warehouse } from '@ui/gen/v1alpha1/generated_pb';

import { EditWarehouseModal } from './edit-warehouse-modal';

export const WarehouseActions = ({ warehouse }: { warehouse: Warehouse }) => {
  const { name: projectName, warehouseName } = useParams();
  const navigate = useNavigate();
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();

  const { mutate, isPending: isLoadingDelete } = useMutation(deleteWarehouse, {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: createConnectQueryKey(listWarehouses) })
  });

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  const onDelete = () => {
    confirm({
      onOk: () => {
        mutate({ name: warehouse.metadata?.name, project: projectName });
        onClose();
      },
      title: 'Are you sure you want to delete Warehouse?'
    });
  };

  const { show: showEditWarehouseModal } = useModal((p) =>
    warehouseName && projectName ? (
      <EditWarehouseModal {...p} warehouseName={warehouseName} projectName={projectName} />
    ) : null
  );

  return (
    <Space size={16}>
      <Button
        type='default'
        icon={<FontAwesomeIcon icon={faPen} size='1x' />}
        onClick={() => showEditWarehouseModal()}
      >
        Edit
      </Button>
      <Button
        danger
        type='text'
        icon={<FontAwesomeIcon icon={faTrash} size='1x' />}
        onClick={onDelete}
        loading={isLoadingDelete}
        size='small'
      >
        Delete
      </Button>
    </Space>
  );
};
