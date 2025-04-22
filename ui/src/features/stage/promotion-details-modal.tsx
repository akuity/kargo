import { toJson } from '@bufbuild/protobuf';
import { useMutation } from '@connectrpc/connect-query';
import { faFileLines, faShoePrints, faStopCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Descriptions, message, Modal, Tabs } from 'antd';

import { ManifestPreview } from '@ui/features/common/manifest-preview';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { canAbortPromotion } from '@ui/features/stage/utils/promotion';
import { abortPromotion } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Promotion, PromotionSchema } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { PromotionSteps } from './promotion-steps';

export const PromotionDetailsModal = ({
  promotion,
  hide,
  visible,
  project
}: {
  promotion: Promotion;
  project: string;
} & ModalProps) => {
  const abortPromotionMutation = useMutation(abortPromotion, {
    onSuccess: () =>
      // Abort promotion annotates the Promotion resource and then controller acts
      message.success({
        content: `Abort Promotion ${promotion.metadata?.name} requested successfully.`
      })
  });

  const confirmAbortRequest = () =>
    Modal.confirm({
      width: '656px',
      icon: <FontAwesomeIcon icon={faStopCircle} className='text-lg text-red-500 mr-5' />,
      title: 'Abort Promotion Request',
      onOk: () => abortPromotionMutation.mutate({ project, name: promotion?.metadata?.name }),
      okText: 'Abort',
      okButtonProps: {
        danger: true
      },
      content: (
        <Descriptions
          size='small'
          className='mt-2'
          column={1}
          bordered
          items={[
            {
              key: 'name',
              label: 'Name',
              children: promotion.metadata?.name
            },
            {
              key: 'date',
              label: 'Start Date',
              children: timestampDate(promotion.metadata?.creationTimestamp)?.toString()
            }
          ]}
        />
      )
    });

  return (
    <Modal
      title='Promotion Details'
      open={visible}
      width='800px'
      okButtonProps={{ hidden: true }}
      onOk={hide}
      onCancel={hide}
    >
      <Tabs
        defaultActiveKey='1'
        tabBarExtraContent={
          canAbortPromotion(promotion) && (
            <Button
              danger
              icon={<FontAwesomeIcon icon={faStopCircle} className='text-sm' />}
              onClick={confirmAbortRequest}
              size='small'
            >
              Abort
            </Button>
          )
        }
      >
        {promotion.spec?.steps && (
          <Tabs.TabPane tab='Steps' key='1' icon={<FontAwesomeIcon icon={faShoePrints} />}>
            <PromotionSteps promotion={promotion} />
          </Tabs.TabPane>
        )}
        <Tabs.TabPane tab='YAML' key='2' icon={<FontAwesomeIcon icon={faFileLines} />}>
          <ManifestPreview object={toJson(PromotionSchema, promotion)} height='500px' />
        </Tabs.TabPane>
      </Tabs>
    </Modal>
  );
};
