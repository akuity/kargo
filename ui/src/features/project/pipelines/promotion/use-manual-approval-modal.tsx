import { useMutation } from '@connectrpc/connect-query';
import { Alert, Typography } from 'antd';
import { generatePath, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { approveFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

type CallbackProps = {
  projectName: string;
  stage: string;
  freight: string;
  onClose?: () => void;
};

export const useManualApprovalModal = () => {
  const confirm = useConfirmModal();
  const manualApproveActionMutation = useMutation(approveFreight);
  const navigate = useNavigate();

  return ({ freight, onClose, projectName, stage }: CallbackProps) =>
    confirm({
      title: 'Manual approval required',
      okText: 'Approve',
      width: 600,
      content: (
        <>
          <Alert
            banner
            type='error'
            message={
              <>
                The selected Freight does not meet criteria for Promotion to this Stage, such as
                successful Promotion to and verification in upstream Stages. Manually approving the
                Freight for Promotion to this Stage will dismiss bypass the unmet criteria.
              </>
            }
          />
          <Typography.Paragraph className='mt-2'>
            Do you want to manually approve?
          </Typography.Paragraph>
        </>
      ),
      onCancel: onClose,
      onOk: async () =>
        await manualApproveActionMutation.mutateAsync(
          {
            stage,
            project: projectName,
            name: freight
          },
          {
            onSuccess: () => {
              onClose?.();
              navigate(
                generatePath(paths.promote, {
                  name: projectName,
                  freight: freight,
                  stage
                })
              );
            }
          }
        )
    });
};
