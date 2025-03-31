import { useMutation } from '@connectrpc/connect-query';
import { Modal } from 'antd';
import { useParams } from 'react-router-dom';

import { deleteRole } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

export const DeleteRoleModal = ({
  name,
  hide,
  onSuccess
}: {
  name: string;
  hide: () => void;
  onSuccess: () => void;
}) => {
  const { name: project } = useParams();
  const { mutate } = useMutation(deleteRole, {
    onSuccess: () => {
      hide();
      onSuccess();
    }
  });

  return (
    <Modal
      title='Delete Role'
      visible
      onOk={() => {
        mutate({ project, name });
      }}
      onCancel={hide}
    >
      <p>Are you sure you want to delete the role {name}?</p>
    </Modal>
  );
};
