import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Divider, Drawer, Empty, Typography } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status-icon/health-status-icon';
import { AvailableStates } from '@ui/features/stage/available-states';
import { Subscriptions } from '@ui/features/stage/subscriptions';
import {
  deleteStage,
  getStage,
  listStages
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { ButtonIcon, LoadingState } from '../common';
import { useConfirmModal } from '../common/confirm-modal/use-confirm-modal';

export const StageDetails = () => {
  const { name: projectName, stageName } = useParams();
  const confirm = useConfirmModal();
  const navigate = useNavigate();

  const { data, isLoading, refetch } = useQuery({
    ...getStage.useQuery({ project: projectName, name: stageName }),
    enabled: !!stageName
  });
  const queryClient = useQueryClient();

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  const { mutate, isLoading: isLoadingDelete } = useMutation({
    ...deleteStage.useMutation(),
    onSuccess: () => queryClient.invalidateQueries(listStages.getQueryKey({ project: projectName }))
  });

  const onDelete = () => {
    confirm({
      onOk: () => {
        mutate({ name: data?.stage?.metadata?.name, project: projectName });
        onClose();
      },
      title: 'Are you sure you want to delete Stage?'
    });
  };

  return (
    <Drawer open={!!stageName} onClose={onClose} width={'80%'} closable={false}>
      {isLoading && <LoadingState />}
      {!isLoading && !data?.stage && <Empty description='Stage not found' />}
      {data?.stage && (
        <>
          <div className='flex items-center justify-between'>
            <div className='flex gap-1 items-start'>
              <HealthStatusIcon
                health={data.stage.status?.currentState?.health}
                style={{ marginRight: '10px', marginTop: '10px' }}
              />
              <div>
                <Typography.Title level={1} style={{ margin: 0 }}>
                  {data.stage.metadata?.name}
                </Typography.Title>
                <Typography.Text type='secondary'>{projectName}</Typography.Text>
              </div>
            </div>
            <Button
              danger
              type='text'
              icon={<ButtonIcon icon={faTrash} size='1x' />}
              onClick={onDelete}
              loading={isLoadingDelete}
            >
              Delete
            </Button>
          </div>
          <Divider style={{ marginTop: '1em' }} />

          <div className='flex flex-col gap-8'>
            <Subscriptions
              subscriptions={data.stage.spec?.subscriptions}
              projectName={projectName}
            />
            <AvailableStates stage={data.stage} onSuccess={refetch} />
          </div>
        </>
      )}
    </Drawer>
  );
};
