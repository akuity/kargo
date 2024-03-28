import { useMutation } from '@connectrpc/connect-query';
import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Modal } from 'antd';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { deleteCredentials } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

type Props = ModalComponentProps & {
  project: string;
  name: string;
  onSuccess: () => void;
  editing?: boolean;
};

export const DeleteCredentialsModal = ({ project, name, onSuccess, ...props }: Props) => {
  const { mutate } = useMutation(deleteCredentials, {
    onSuccess: () => {
      props.hide();
      onSuccess();
    }
  });

  return (
    <Modal
      onCancel={props.hide}
      onOk={() => {
        mutate({ project, name });
      }}
      title={
        <div className='flex items-center'>
          <FontAwesomeIcon icon={faTrash} className='mr-2' />
          Delete Credentials
        </div>
      }
      okText='Yes, Delete'
      okType='danger'
      {...props}
    >
      <p>
        Are you sure you want to delete the credentials <b>{name}</b>?
      </p>
    </Modal>
  );
};
