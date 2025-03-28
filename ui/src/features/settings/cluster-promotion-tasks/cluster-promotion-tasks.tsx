import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Table } from 'antd';
import { format } from 'date-fns';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import { ClusterPromotionTaskManifestsGen } from '@ui/features/utils/manifest-generator';
import {
  deleteResource,
  listClusterPromotionTasks
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ClusterPromotionTask } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { CreateClusterPromotionTaskModal } from './create-cluster-promotion-task';
import { EditClusterPromotionTaskModal } from './edit-cluster-promotion-task-modal';

export const ClusterPromotionTasks = () => {
  const confirm = useConfirmModal();

  const listClusterPromotionTasksQuery = useQuery(listClusterPromotionTasks);

  const deleteClusterPromotionTaskMutation = useMutation(deleteResource);

  const clusterPromotionTasksModal = useModal();

  const onAddClusterPromotionTaskModalOpen = () =>
    clusterPromotionTasksModal.show((p) => <CreateClusterPromotionTaskModal {...p} />);

  const onDeleteClusterPromotionTaskModalOpen = (clusterPromotionTask: ClusterPromotionTask) =>
    confirm({
      title: (
        <Flex align='center'>
          <FontAwesomeIcon icon={faTrash} className='mr-2' />
          Delete Cluster Promotion Task
        </Flex>
      ),
      content: (
        <p>
          Are you sure you want to delete ClusterPromotionTask{' '}
          <b>{clusterPromotionTask?.metadata?.name}</b>?
        </p>
      ),
      onOk: () => {
        const manifest = new TextEncoder().encode(
          ClusterPromotionTaskManifestsGen.v1alpha1(clusterPromotionTask)
        );
        deleteClusterPromotionTaskMutation.mutate(
          { manifest },
          { onSuccess: () => listClusterPromotionTasksQuery.refetch() }
        );
      },
      hide: () => {}
    });

  const onEditClusterPromotionTaskModalOpen = (clusterPromotionTask: ClusterPromotionTask) =>
    clusterPromotionTasksModal.show((p) => (
      <EditClusterPromotionTaskModal {...p} clusterPromotionTask={clusterPromotionTask} />
    ));

  return (
    <div className='p-4'>
      <Table<ClusterPromotionTask>
        dataSource={listClusterPromotionTasksQuery.data?.clusterPromotionTasks}
        loading={listClusterPromotionTasksQuery.isFetching}
        locale={{
          emptyText: (
            <>
              This project does not have any Promotion Tasks. Read more about ClusterPromotionTasks{' '}
              <a
                href='https://docs.kargo.io/user-guide/reference-docs/promotion-tasks/#defining-a-global-promotion-task'
                target='_blank'
              >
                here
              </a>
              .
            </>
          )
        }}
        columns={[
          {
            title: 'Creation Date',
            width: 200,
            render: (_, template) => {
              const date = timestampDate(template.metadata?.creationTimestamp);

              return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
            }
          },
          {
            title: 'Name',
            render: (_, r) => r.metadata?.name
          },
          {
            title: (
              <div className='text-right'>
                <Button
                  type='primary'
                  className='ml-auto text-xs font-semibold'
                  icon={<FontAwesomeIcon icon={faPlus} />}
                  onClick={onAddClusterPromotionTaskModalOpen}
                >
                  ADD CLUSTER PROMOTION TASK
                </Button>
              </div>
            ),
            render: (_, template) => (
              <Flex gap={8} justify='end'>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} />}
                  onClick={() => onEditClusterPromotionTaskModalOpen(template)}
                >
                  Edit
                </Button>
                <Button
                  icon={<FontAwesomeIcon icon={faTrash} />}
                  danger
                  onClick={() => onDeleteClusterPromotionTaskModalOpen(template)}
                >
                  Delete
                </Button>
              </Flex>
            )
          }
        ]}
      />
    </div>
  );
};
