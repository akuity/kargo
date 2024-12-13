import { createConnectQueryKey, useMutation } from '@connectrpc/connect-query';
import {
  faExclamationCircle,
  faPen,
  faRedo,
  faRefresh,
  faTrash
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Space } from 'antd';
import React from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  abortVerification,
  deleteStage,
  queryFreight,
  refreshStage,
  reverify
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

import { useConfirmModal } from '../common/confirm-modal/use-confirm-modal';
import { useModal } from '../common/modal/use-modal';
import { currentFreightHasVerification } from '../common/utils';

import { EditStageModal } from './edit-stage-modal';

export const StageActions = ({
  stage,
  verificationRunning
}: {
  stage: Stage;
  verificationRunning?: boolean;
}) => {
  const { name: projectName, stageName } = useParams();
  const navigate = useNavigate();
  const confirm = useConfirmModal();
  const queryClient = useQueryClient();
  const [shouldRefetchFreights, setShouldRefetchFreights] = React.useState(false);

  const { mutate, isPending: isLoadingDelete } = useMutation(deleteStage);
  const { mutate: refresh, isPending: isRefreshLoading } = useMutation(refreshStage);

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  const onDelete = () => {
    confirm({
      onOk: () => {
        mutate({ name: stage.metadata?.name, project: projectName });
        onClose();
      },
      title: 'Are you sure you want to delete Stage?',
      hide: () => {}
    });
  };

  const onRefresh = () => refresh({ name: stageName, project: projectName });

  const { show: showEditStageModal } = useModal((p) =>
    stageName && projectName ? (
      <EditStageModal {...p} stageName={stageName} projectName={projectName} />
    ) : null
  );

  // Once the Refresh process is done, refetch Freight list
  React.useEffect(() => {
    const refreshRequest = stage?.metadata?.annotations['kargo.akuity.io/refresh'];
    const refreshStatus = stage?.status?.lastHandledRefresh;
    if (refreshRequest !== undefined && refreshRequest !== refreshStatus) {
      setShouldRefetchFreights(true);
    }

    if (refreshRequest === refreshStatus && shouldRefetchFreights) {
      queryClient.invalidateQueries({ queryKey: createConnectQueryKey(queryFreight) });
      setShouldRefetchFreights(false);
    }
  }, [stage, shouldRefetchFreights]);

  const { mutate: reverifyStage, isPending } = useMutation(reverify);
  const { mutate: abortVerificationAction } = useMutation(abortVerification);

  const verificationEnabled = stage?.spec?.verification;

  return (
    <Space size={16}>
      {currentFreightHasVerification(stage) && (
        <>
          {verificationEnabled && (
            <Button
              icon={<FontAwesomeIcon icon={faRedo} spin={isPending} />}
              disabled={isPending || verificationRunning}
              onClick={() => {
                reverifyStage({ project: projectName, stage: stageName });
              }}
            >
              Reverify
            </Button>
          )}
          <Button
            type='default'
            disabled={!verificationRunning}
            icon={<FontAwesomeIcon icon={faExclamationCircle} size='1x' />}
            onClick={() => abortVerificationAction({ project: projectName, stage: stageName })}
          >
            Abort Verification
          </Button>
        </>
      )}

      <Button
        type='default'
        icon={<FontAwesomeIcon icon={faPen} size='1x' />}
        onClick={() => showEditStageModal()}
      >
        Edit
      </Button>
      <Button
        type='default'
        icon={<FontAwesomeIcon icon={faRefresh} size='1x' />}
        onClick={onRefresh}
        loading={
          isRefreshLoading ||
          (!!stage?.metadata?.annotations['kargo.akuity.io/refresh'] &&
            stage?.metadata?.annotations?.['kargo.akuity.io/refresh'] !==
              stage?.status?.lastHandledRefresh)
        }
      >
        Refresh
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
