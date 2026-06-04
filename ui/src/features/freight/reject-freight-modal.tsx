import { faBan } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Input, Modal, message } from 'antd';
import { useState } from 'react';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { getAlias } from '@ui/features/common/utils';
import type { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { useRejectFreight } from '@ui/gen/api/v2/core/core';

type Props = ModalComponentProps & {
  freight: Freight;
  project: string;
  onSubmit: () => void;
};

export const RejectFreightModal = ({ freight, project, onSubmit, hide, ...props }: Props) => {
  const [reason, setReason] = useState('');
  const alias = getAlias(freight);
  const freightNameOrAlias = freight.metadata?.name || alias || '';

  const { mutate: rejectAction, isPending } = useRejectFreight({
    mutation: {
      onSuccess: () => {
        message.success('Freight rejected');
        onSubmit();
      },
      onError: (err) => {
        message.error(err?.toString());
      }
    }
  });

  return (
    <Modal
      {...props}
      title={
        <>
          <FontAwesomeIcon icon={faBan} className='mr-2' />
          Reject Freight
        </>
      }
      onCancel={hide}
      okText='Reject'
      okButtonProps={{ danger: true, loading: isPending }}
      onOk={() =>
        rejectAction({
          project,
          freightNameOrAlias,
          data: { reason }
        })
      }
    >
      <div className='mb-4'>
        <div className='text-xs font-semibold uppercase'>Freight</div>
        <div className='font-mono'>{alias || freight.metadata?.name}</div>
      </div>
      <Input.TextArea
        value={reason}
        maxLength={1024}
        showCount
        rows={4}
        placeholder='Reason'
        onChange={(event) => setReason(event.target.value)}
      />
    </Modal>
  );
};
