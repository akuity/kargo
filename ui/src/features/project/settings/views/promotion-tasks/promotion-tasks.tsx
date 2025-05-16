import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faPencil, faPlus, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Table } from 'antd';
import { format } from 'date-fns';
import { useParams } from 'react-router-dom';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import { promotionTaskManifestsGen } from '@ui/features/utils/manifest-generator';
import {
  deleteResource,
  listPromotionTasks
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { PromotionTask } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { CreatePromotionTaskModal } from './create-promotion-task';
import { EditPromotionTaskModal } from './edit-promotion-task-modal';

export const PromotionTasks = () => {
  const { name } = useParams();
  const confirm = useConfirmModal();

  const listPromotionTasksQuery = useQuery(listPromotionTasks, { project: name });

  const deletePromotionTaskMutation = useMutation(deleteResource);

  const promotionTasksModal = useModal();

  const onAddPromotionTaskModalOpen = () =>
    promotionTasksModal.show((p) => <CreatePromotionTaskModal {...p} namespace={name || ''} />);

  const onDeletePromotionTaskModalOpen = (promotionTask: PromotionTask) =>
    confirm({
      title: (
        <Flex align='center'>
          <FontAwesomeIcon icon={faTrash} className='mr-2' />
          Delete Promotion Task
        </Flex>
      ),
      content: (
        <p>
          Are you sure you want to delete PromotionTask <b>{promotionTask?.metadata?.name}</b>?
        </p>
      ),
      onOk: () => {
        const manifest = new TextEncoder().encode(
          promotionTaskManifestsGen.v1alpha1(promotionTask)
        );
        deletePromotionTaskMutation.mutate(
          { manifest },
          { onSuccess: () => listPromotionTasksQuery.refetch() }
        );
      },
      hide: () => {}
    });

  const onEditPromotionTaskModalOpen = (promotionTask: PromotionTask) =>
    promotionTasksModal.show((p) => (
      <EditPromotionTaskModal {...p} promotionTask={promotionTask} />
    ));

  return (
    <Card
      title='Promotion Tasks'
      type='inner'
      className='min-h-full'
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} />} onClick={onAddPromotionTaskModalOpen}>
          Add Promotion Task
        </Button>
      }
    >
      <Table<PromotionTask>
        className='my-2'
        dataSource={listPromotionTasksQuery.data?.promotionTasks}
        loading={listPromotionTasksQuery.isFetching}
        locale={{
          emptyText: (
            <>
              This project does not have any Promotion Tasks. Read more about PromotionTasks{' '}
              <a
                href='https://docs.kargo.io/user-guide/reference-docs/promotion-tasks/'
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
            render: (_, template) => (
              <Flex gap={8} justify='end'>
                <Button
                  icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
                  onClick={() => onEditPromotionTaskModalOpen(template)}
                  color='default'
                  variant='filled'
                  size='small'
                >
                  Edit
                </Button>
                <Button
                  icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
                  onClick={() => onDeletePromotionTaskModalOpen(template)}
                  color='danger'
                  variant='filled'
                  size='small'
                >
                  Delete
                </Button>
              </Flex>
            )
          }
        ]}
      />
    </Card>
  );
};
