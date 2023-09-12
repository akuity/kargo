import { faArrowTurnUp } from '@fortawesome/free-solid-svg-icons';
import { useMutation } from '@tanstack/react-query';
import { Modal, Typography, message } from 'antd';
import React from 'react';

import { ButtonIcon } from '@ui/features/common';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { promoteSubscribers } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/v1alpha1/types_pb';

type Props = ModalProps & {
  stages: Stage[];
  selectedStage: Stage;
};

export const PromoteSubscribersModal = ({ stages, selectedStage, visible, hide }: Props) => {
  const { mutate, isLoading: isLoadingPromoteSubscribers } = useMutation({
    ...promoteSubscribers.useMutation(),
    onError: (err) => {
      message.error(err?.toString());
    },
    onSuccess: () => {
      message.success(
        `All subscribers of "${selectedStage?.metadata?.name}" stage have been promoted.`
      );
      hide();
    }
  });

  const subscribersByStage = React.useMemo(
    () =>
      stages.reduce((acc, stage) => {
        stage?.spec?.subscriptions?.upstreamStages.forEach((item) => {
          const items = acc[item.name] || [];
          acc[item.name] = [...items, stage];
        });

        return acc;
      }, {} as { [key: string]: Stage[] }),
    [stages]
  );

  const promote = () => {
    mutate({
      stage: selectedStage.metadata?.name || '',
      project: selectedStage.metadata?.namespace || '',
      freight: selectedStage.status?.currentFreight?.id || ''
    });
  };

  return (
    <Modal
      open={visible}
      onCancel={hide}
      okButtonProps={{
        icon: <ButtonIcon icon={faArrowTurnUp} />,
        loading: isLoadingPromoteSubscribers
      }}
      onOk={promote}
      width={'500px'}
      closable={false}
      okText='Promote'
    >
      <div className='text-xl font-semibold mb-4'>Promote Subscribers</div>
      <div>
        Are you sure you want to promote all subscribers of stage{' '}
        <span className='font-bold'>{selectedStage?.metadata?.name}</span>?
      </div>
      <div className='mt-4'>
        <div className='mb-2'>
          This action will promote the following stages to Freight ID{' '}
          <Typography.Text code>{selectedStage?.status?.currentFreight?.id}</Typography.Text>.
        </div>
        {(subscribersByStage[selectedStage?.metadata?.name || ''] || []).map((item) => (
          <div key={item.metadata?.uid} className='font-bold text-lg'>
            - {item.metadata?.name}
          </div>
        ))}
      </div>
    </Modal>
  );
};
