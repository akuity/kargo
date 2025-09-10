import { useMutation } from '@connectrpc/connect-query';
import { Alert } from 'antd';
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
      content: (
        <Alert
          banner
          message='The selected freight has not been allowed for promotion to this stage. If you confirm, we will manually approve it. Do you want to proceed?'
        />
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
